package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type AdminClient struct {
	client  http.Client
	token   string
	baseURL *url.URL
}

func NewAdminClient(token string) (*AdminClient, error) {
	parsedUrl, _ := url.Parse("https://api.atlassian.com/")
	c := &AdminClient{
		client:  *http.DefaultClient,
		token:   token,
		baseURL: parsedUrl,
	}

	return c, nil
}

func (c *AdminClient) NewRequestWithContext(ctx context.Context, method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// Relative URLs should be specified without a preceding slash since baseURL will have the trailing slash
	rel.Path = strings.TrimLeft(rel.Path, "/")

	u := c.baseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	return req, nil
}

func (c *AdminClient) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if c := resp.StatusCode; !(200 <= c && c <= 299) {
		return nil, fmt.Errorf("request failed. Please analyze the request body for more details. Status code: %d", c)

	}

	if v != nil {
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(v)
	}

	return resp, err
}
