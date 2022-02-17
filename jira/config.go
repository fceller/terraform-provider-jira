package jira

import (
	jira "github.com/andygrunwald/go-jira"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"log"
)

type Config struct {
	jiraClient  *jira.Client
	adminClient *AdminClient
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
