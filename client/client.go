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
)

type Client struct {
	Username string
	Password string
	Host     string
	Port     int
	token    string
}

type Response struct {
	Status  string
	Message string `json:"msg"`
	Data    json.RawMessage
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

	j := &Response{}
	err = json.Unmarshal(b, j)
	if err != nil {
		return "", err
	}
	if j.Status != "ok" {
		msg := j.Message
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

	// get data
	tmp := make(map[string]interface{})
	err = json.Unmarshal(b, &tmp)
	if err != nil {
		return "", err
	}

	// convert back to json string with indentation
	pretty, err := json.MarshalIndent(tmp, "", "  ")
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

	j := &Response{}
	err = json.Unmarshal(b, j)
	if err != nil {
		return err
	}

	if j.Status != "ok" {
		return fmt.Errorf(j.Message)
	}

	err = json.Unmarshal(j.Data, &bc.token)
	if err != nil {
		return err
	}

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

	j := &Response{}
	err = json.Unmarshal(b, j)
	if err != nil {
		return err
	}
	if j.Status != "ok" {
		return errors.New(j.Message)
	}

	var ticket string
	err = json.Unmarshal(j.Data, &ticket)
	fmt.Println(ticket)
	if err != nil {
		return err
	}

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

	j2 := &Response{}
	err = json.Unmarshal(b2, j2)
	if err != nil {
		return err
	}
	if j2.Status != "ok" {
		return errors.New(j2.Message)
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
		j := &Response{}
		err = json.Unmarshal(b, j)
		if err != nil {
			return err
		}
		return fmt.Errorf(j.Message)
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
