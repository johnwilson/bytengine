package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/bitly/go-simplejson"
)

type Client struct {
	Username string
	Password string
	Host     string
	Port     int
	token    string
}

func (bc *Client) Exec(cmd string, retry int) (string, error) {
	_url := fmt.Sprintf("http://%s:%d/bfs/query", bc.Host, bc.Port)
	v := url.Values{}
	v.Set("token", bc.token)
	v.Set("query", cmd)
	rep, err := http.PostForm(_url, v)
	if err != nil {
		return "", err
	}
	defer rep.Body.Close()
	b, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return "", err
	}

	j, err := simplejson.NewJson(b)
	if err != nil {
		return "", err
	}
	if j.Get("status").MustString("") != "ok" {
		msg := j.Get("msg").MustString("")
		if msg == "invalid auth token" && retry < 4 {
			retry += 1
			err = bc.newToken()
			if err != nil {
				return "", err
			}
			return bc.Exec(cmd, retry)
		} else {
			return "", errors.New(msg)
		}
	}

	data, err := j.Map()
	if err != nil {
		return "", err
	}
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(pretty), nil
}

func (bc *Client) Connect(host string, port int) error {
	_url := fmt.Sprintf("http://%s:%d/", host, port)
	rep, err := http.Get(_url)
	if err != nil {
		return err
	}
	defer rep.Body.Close()
	if rep.StatusCode != 200 {
		return fmt.Errorf("connection to Bytengine server failed.")
	}

	bc.Host = host
	bc.Port = port
	return nil
}

func (bc *Client) Login(username, password string) error {
	bc.Username = username
	bc.Password = password
	err := bc.newToken()
	if err != nil {
		// reset username/password
		bc.Username = ""
		bc.Password = ""
		return err
	}
	return nil
}

func (bc *Client) newToken() error {
	_url := fmt.Sprintf("http://%s:%d/bfs/token", bc.Host, bc.Port)
	v := url.Values{}
	v.Set("username", bc.Username)
	v.Set("password", bc.Password)
	rep, err := http.PostForm(_url, v)
	if err != nil {
		return err
	}
	defer rep.Body.Close()
	b, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return err
	}

	j, err := simplejson.NewJson(b)
	if err != nil {
		return err
	}
	if j.Get("status").MustString("") != "ok" {
		msg := j.Get("msg").MustString("")
		return fmt.Errorf(msg)
	}

	bc.token = j.Get("data").MustString("")
	return nil
}

func (bc Client) WriteBytes(db, remotefile, localfile string) error {
	// get upload ticket
	_url := fmt.Sprintf("http://%s:%d/bfs/uploadticket", bc.Host, bc.Port)
	v := url.Values{}
	v.Set("token", bc.token)
	v.Set("database", db)
	v.Set("path", remotefile)
	rep, err := http.PostForm(_url, v)
	if err != nil {
		return err
	}
	defer rep.Body.Close()
	b, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		return err
	}

	j, err := simplejson.NewJson(b)
	if err != nil {
		return err
	}
	if j.Get("status").MustString("") != "ok" {
		msg := j.Get("msg").MustString("")
		return errors.New(msg)
	}
	ticket := j.Get("data").MustString("")

	// upload file
	buffer := &bytes.Buffer{}
	bodywriter := multipart.NewWriter(buffer)
	filewriter, err := bodywriter.CreateFormFile("file", path.Base(localfile))
	if err != nil {
		return err
	}

	fp, err := os.Open(localfile)
	if err != nil {
		return err
	}
	_, err = io.Copy(filewriter, fp)
	if err != nil {
		return err
	}
	ctype := bodywriter.FormDataContentType()
	bodywriter.Close()

	_url = fmt.Sprintf("http://%s:%d/bfs/writebytes/%s", bc.Host, bc.Port, ticket)
	rep2, err := http.Post(_url, ctype, buffer)
	if err != nil {
		return err
	}

	// get response
	defer rep2.Body.Close()
	b2, err := ioutil.ReadAll(rep2.Body)
	if err != nil {
		return err
	}

	j2, err := simplejson.NewJson(b2)
	if err != nil {
		return err
	}
	if j2.Get("status").MustString("") != "ok" {
		msg := j2.Get("msg").MustString("")
		return errors.New(msg)
	}
	return nil
}

func (bc Client) ReadBytes(db, remotefile, localfile string) error {
	_url := fmt.Sprintf("http://%s:%d/bfs/readbytes", bc.Host, bc.Port)
	v := url.Values{}
	v.Set("token", bc.token)
	v.Set("database", db)
	v.Set("path", remotefile)
	rep, err := http.PostForm(_url, v)
	if err != nil {
		return err
	}
	defer rep.Body.Close()
	if rep.StatusCode != 200 {
		b, err := ioutil.ReadAll(rep.Body)
		if err != nil {
			return err
		}
		j, err := simplejson.NewJson(b)
		if err != nil {
			return err
		}
		msg := j.Get("msg").MustString("")
		return fmt.Errorf(msg)
	}
	fp, err := os.Create(localfile)
	defer fp.Close()

	if err != nil {
		return err
	}
	_, err = io.Copy(fp, rep.Body)
	if err != nil {
		return err
	}
	return nil
}

func NewClient() *Client {
	return &Client{
		Host: "localhost",
		Port: 8080,
	}
}
