package questionsclient

import "strings"

type Client struct {
	baseURL string
}

func New(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}
