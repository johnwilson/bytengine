package modules

import (
	"fmt"
	"net/http"
	"net/url"
	"io/ioutil"
	"encoding/json"
	"time"
)

type FunctionManager struct {
	config ConfigRemote
}

func NewFunctionManager(c ConfigRemote) *FunctionManager {
	val := FunctionManager{ config: c	}
	return &val
}

func (fm *FunctionManager) Exec(fn string, args map[string]interface{}) (interface{}, error) {
	var timeout = time.Second * time.Duration(fm.config.Exec.Timeout)
	var _url = fm.config.Exec.Url + fn

	var tr = &http.Transport {
		ResponseHeaderTimeout: timeout,
	}
	var client = &http.Client{ Transport: tr }

	var v = url.Values{}
	j, err := json.Marshal(args)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	v.Set("args",string(j))
	
	resp, err := client.PostForm(_url,v)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var txt = string(body)
	return txt, nil
}

func (fm *FunctionManager) Pipe(fn string, data interface{}, args map[string]interface{}) (interface{}, error) {
	var timeout = time.Second * time.Duration(fm.config.Pipe.Timeout)
	var _url = fm.config.Pipe.Url + fn

	var tr = &http.Transport {
		ResponseHeaderTimeout: timeout,
	}
	var client = &http.Client{ Transport: tr }

	var v = url.Values{}
	bArgs, err := json.Marshal(args)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	v.Set("args",string(bArgs))
	bData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	v.Set("data",string(bData))
	
	resp, err := client.PostForm(_url,v)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var txt = string(body)
	return txt, nil
}