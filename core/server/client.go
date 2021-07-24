package server

import (
	"errors"
	"io"
	"net/http"
)

// Client is a client interface that enables to write custom behaviours to servers
type Client interface {
	Get(url string) (*http.Response, error)
	Post(url string, body io.Reader) (*http.Response, error)
	Do(*http.Request) (*http.Response, error)
	SetRequest(method, url string, body io.Reader) (*http.Request, error)
}

type flag struct {
	requestError bool
	doError      bool
}

type client struct {
	client *http.Client
	*flag
}

// New Creates a new Client
func New(config ...*flag) Client {
	var f *flag
	if len(config) == 0 {
		f = &flag{}
	} else {
		f = config[0]
	}
	return &client{
		client: &http.Client{},
		flag:   f,
	}
}

var (
	// ErrorNilURL the url is missing
	ErrorNilURL = errors.New("The provided URL can't be empty")
	// errorRequest is used to test code behaviour in case setting the requests fails
	errorRequest = errors.New("forced error setting request")
)

func (c *client) Get(url string) (*http.Response, error) {
	if url == "" {
		return nil, ErrorNilURL
	}

	req, err := c.SetRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func (c *client) Post(url string, body io.Reader) (*http.Response, error) {
	if url == "" {
		return nil, ErrorNilURL
	}

	req, err := c.SetRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-type", "application/json")

	return c.Do(req)
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	if !c.flag.doError {
		req.URL.Scheme = "https"
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *client) SetRequest(method, url string, body io.Reader) (*http.Request, error) {
	if c.flag.requestError {
		return nil, errorRequest
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	return req, nil
}
