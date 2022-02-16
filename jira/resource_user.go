package jira

import (
	"context"
	"fmt"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

type RawUser struct {
	AccountId   string `json:"accountId" structs:"accountId"`
	AccountType string `json:"accountType,omitempty" structs:"accountType,omitempty"`
	DisplayName string `json:"displayName,omitempty" structs:"displayName,omitempty"`
}

type RawUsers struct {
	Total int       `json:"total" structs:"total"`
	Users []RawUser `json:"users" structs:"users"`
}

type RawSearch struct {
	Users RawUsers `json:"users" structs:"users"`
}

// resourceUser is used to define a JIRA issue
func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceUserCreate,
		Read:   resourceUserRead,
		Delete: resourceUserDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"account_id": {
				Description: "The Atlassian account id.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "The name of the user.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"email": {
				Description: "The email address of the user.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func getUserByKey(client *jira.Client, key string) (*jira.User, *jira.Response, error) {
	apiEndpoint := fmt.Sprintf("%s?accountId=%s", userAPIEndpoint, key)
	req, err := client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}

	user := new(jira.User)
	resp, err := client.Do(req, user)
	if err != nil {
		return nil, resp, jira.NewJiraError(resp, err)
	}
	return user, resp, nil
}

func deleteUserByKey(client *jira.Client, key string) (*jira.Response, error) {
	apiEndpoint := fmt.Sprintf("%s?accountId=%s", userAPIEndpoint, key)
	req, err := client.NewRequest("DELETE", apiEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req, nil)
	if err != nil {
		return resp, jira.NewJiraError(resp, err)
	}
	return resp, nil
}

// resourceUserCreate creates a new jira user using the jira api
func resourceUserCreate(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	user := new(jira.User)
	user.DisplayName = d.Get("display_name").(string)
	user.EmailAddress = d.Get("email").(string)

	createdUser, _, err := config.jiraClient.User.Create(user)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	d.SetId(createdUser.AccountID)

	err = resourceUserRead(d, m)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	d.Set("email", user.EmailAddress)

	return nil
}

// resourceUserRead reads issue details using jira api
func resourceUserRead(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)
	id := d.Id()

	if strings.Contains(id, "@") {
		apiEndpoint := fmt.Sprintf("/rest/api/2/groupuserpicker?query=%s&showAvatar=false&excludedConnectAddons=true", id)
		req, _ := config.jiraClient.NewRequest("GET", apiEndpoint, nil)
		search := new(RawSearch)
		_, err := config.jiraClient.Do(req, &search)

		if err == nil {
			tflog.Info(context.Background(), "RESULT: "+string(search.Users.Total))
		} else {
			return errors.Wrap(err, "getting jira user via search failed")
		}
	} else {
		user, _, err := getUserByKey(config.jiraClient, id)
		if err != nil {
			return errors.Wrap(err, "getting jira user failed")
		}

		d.Set("account_id", user.AccountID)
		d.Set("display_name", user.DisplayName)
	}
	return nil
}

// resourceUserDelete deletes jira issue using the jira api
func resourceUserDelete(d *schema.ResourceData, m interface{}) error {
	config := m.(*Config)

	_, err := deleteUserByKey(config.jiraClient, d.Id())

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	return nil
}
