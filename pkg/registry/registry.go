// Package registry implements an http client for the docker registry.
package registry

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

const DefaultURL = "https://registry.hub.docker.com"

type Client struct {
	URL                string
	Username, Password string
	client             *http.Client
}

func New(c *http.Client) *Client {
	if c == nil {
		c = http.DefaultClient
	}

	return &Client{
		client: c,
	}
}

// Tag tags the given imageID in the repository with the given tag.
func (c *Client) Tag(repo, imageID, tag string) error {
	req, err := c.NewRequest("PUT", "/v1/repositories/"+repo+"/tags/"+tag, strings.NewReader(`"`+imageID+`"`))
	if err != nil {
		return err
	}

	if resp, err := c.client.Do(req); err != nil || resp.StatusCode >= 300 {
		return errors.New("Unsuccessful Request: " + resp.Status)
	}

	return err
}

func (c *Client) NewRequest(method, path string, r io.Reader) (*http.Request, error) {
	url := c.URL
	if url == "" {
		url = DefaultURL
	}
	req, err := http.NewRequest("PUT", url+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(c.Username, c.Password)
	return req, nil
}
