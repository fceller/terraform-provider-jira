package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

type AdminClient struct {
	client  http.Client
	token   string
	baseURL *url.URL
}

type Config struct {
	jiraClient  *jira.Client
	adminClient *AdminClient
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

func (c *AdminClient) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if v != nil {
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(v)
	}

	return resp, err
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer s", c.token))

	return req, nil
}

func (c *Config) createAndAuthenticateClient(d *schema.ResourceData) error {
	log.Printf("[INFO] creating jira client using environment variables")
	jiraClient, err := jira.NewClient(nil, d.Get("url").(string))
	if err != nil {
		return errors.Wrap(err, "creating jira client failed")
	}
	jiraClient.Authentication.SetBasicAuth(d.Get("user").(string), d.Get("password").(string))

	c.jiraClient = jiraClient

	log.Printf("[INFO] creating admin client using environment variables")
	adminClient, err2 := NewAdminClient(d.Get("token").(string))
	if err2 != nil {
		return errors.Wrap(err, "creating admin client failed")
	}

	c.adminClient = adminClient

	return nil
}
