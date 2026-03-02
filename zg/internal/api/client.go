package api

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string, timeoutSecs int) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSecs) * time.Second,
		},
	}
}

func (c *Client) buildURL(path string) string {
	return c.baseURL + path
}

func (c *Client) addAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}

func (c *Client) Get(path string) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) Post(path string, body []byte) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) Put(path string, body []byte) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

func (c *Client) Delete(path string) (*http.Response, error) {
	url := c.buildURL(path)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	c.addAuth(req)
	return c.httpClient.Do(req)
}

// GetBodyBytes reads response body and closes it
func GetBodyBytes(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetBodyString reads response body as string and closes it
func GetBodyString(resp *http.Response) (string, error) {
	body, err := GetBodyBytes(resp)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
