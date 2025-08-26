package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HttpClient struct {
	cli http.Client
}

func NewHttpClient() *HttpClient {
	cli := http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   5,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       time.Second * 90,
			TLSHandshakeTimeout:   time.Second * 10,
			ExpectContinueTimeout: time.Second * 1,
		},
	}
	return &HttpClient{
		cli: cli,
	}
}

func (c *HttpClient) Get(url string, args map[string]any) ([]byte, error) {
	if len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(request)
}

func (c *HttpClient) PostForm(url string, args map[string]any) ([]byte, error) {
	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return c.doRequest(request)
}

func (c *HttpClient) PostJson(url string, args map[string]any) ([]byte, error) {
	jsonData, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	return c.doRequest(request)
}

func (c *HttpClient) GetRequest(url string, args map[string]any) (*http.Request, error) {
	if len(args) > 0 {
		url = url + "?" + c.toValues(args)
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return request, nil
}

func (c *HttpClient) FormRequest(url string, args map[string]any) (*http.Request, error) {
	request, err := http.NewRequest("POST", url, strings.NewReader(c.toValues(args)))
	if err != nil {
		return nil, err
	}
	return request, nil
}

func (c *HttpClient) JsonRequest(url string, args map[string]any) (*http.Request, error) {
	jsonData, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	return request, nil
}

func (c *HttpClient) DoRequest(req *http.Request) ([]byte, error) {
	return c.doRequest(req)
}

func (c *HttpClient) doRequest(req *http.Request) ([]byte, error) {
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	buf := make([]byte, 127)
	body := make([]byte, 0)
	reader := bufio.NewReader(resp.Body)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF || n == 0 {
			break
		}
		body = append(body, buf[:n]...)
		if n < len(buf) {
			break
		}
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)
	return body, nil
}

func (c *HttpClient) toValues(args map[string]any) string {
	if args != nil && len(args) > 0 {
		params := url.Values{}
		for key, value := range args {
			params.Set(key, fmt.Sprintf("%v", value))
		}
		return params.Encode()
	}
	return ""
}
