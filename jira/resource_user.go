package jira

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"account_id": {
				Description: "The Atlassian account id.",
				Type:        schema.TypeString,
				Computed:    true,
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
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
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

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	user := new(jira.User)
	user.DisplayName = d.Get("display_name").(string)
	user.EmailAddress = d.Get("email").(string)

	createdUser, _, err := config.jiraClient.User.Create(user)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Request failed: %s", err.Error()),
		})
		return diags
	}

	d.SetId(createdUser.AccountID)

	diags = resourceUserRead(ctx, d, m)

	if diags.HasError() {
		return diags
	}

	d.Set("email", user.EmailAddress)

	return diags
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()

	if strings.Contains(id, "@") {
		users, _, err := config.jiraClient.User.FindWithContext(ctx, id, nil)

		if err != nil {
			return diag.FromErr(err)
		}

		if len(users) == 1 {
			d.SetId(users[0].AccountID)
			d.Set("account_id", users[0].AccountID)
			d.Set("display_name", users[0].DisplayName)
			d.Set("email", id)
		} else {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("no exact match, found %d users", len(users)),
			})
			return diags
		}
	} else {
		user, _, err := getUserByKey(config.jiraClient, id)
		if err != nil {
			return diag.FromErr(err)
		}

		d.Set("account_id", user.AccountID)
		d.Set("display_name", user.DisplayName)
		d.Set("active", user.Active)
	}

	return diags
}

func RemoveWithContext2(ctx context.Context, m interface{}, groupname string, username string) (*jira.Response, error) {
	config := m.(*Config)

	apiEndpoint := fmt.Sprintf("/rest/api/2/group/user?groupname=%s&accountId=%s", groupname, username)
	req, err := config.jiraClient.NewRequestWithContext(ctx, "DELETE", apiEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := config.jiraClient.Do(req, nil)
	if err != nil {
		jerr := jira.NewJiraError(resp, err)
		return resp, jerr
	}

	return resp, nil
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)
	id := d.Id()

	if active := d.Get("active").(bool); d.HasChange("active") {
		if active {
		} else {
			users, _, err := config.jiraClient.User.GetGroupsWithContext(ctx, id)

			if err != nil {
				return diag.FromErr(err)
			}

			for _, u := range *users {
				RemoveWithContext2(ctx, m, u.Self, id)
			}
		}
	}

	return diags
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	_, err := deleteUserByKey(config.jiraClient, d.Id())

	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

/*

 */

/*
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

apiEndpoint := fmt.Sprintf("/rest/api/2/groupuserpicker?query=%s&showAvatar=false&excludedConnectAddons=true", id)
		req, _ := config.jiraClient.NewRequest("GET", apiEndpoint, nil)
		search := new(RawSearch)
		_, err := config.jiraClient.Do(req, &search)

		if err == nil {
			total := search.Users.Total

			if total != 1 {

			} else {

			}
		} else {
			return diag.FromErr(err)
		}
*/

/*
{
		apiEndpoint := fmt.Sprintf("/rest/api/3/user/groups?accountId=%s", id)
		req, _ := config.jiraClient.NewRequest("GET", apiEndpoint, nil)
		response, _ := config.jiraClient.Do(req, nil)
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		d.Set("groups", string(body))
	}
*/

/*
	id := d.Id()

		state := "enable"
		if !active {
			state = "disable"
		}

		apiEndpoint := fmt.Sprintf("/users/%s/manage/lifecycle/%s", id, state)
		req, _ := config.jiraClient.NewRequest("POST", apiEndpoint, nil)
		_, err := config.jiraClient.Do(req, nil)

		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("Request '%s' failed: %s", apiEndpoint, err.Error()),
			})
			return diags
		}
*/
