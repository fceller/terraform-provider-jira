package jira

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"net/url"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
		apiEndpoint := fmt.Sprintf("/rest/api/2/groupuserpicker?query=%s&showAvatar=false&excludedConnectAddons=true", id)
		req, _ := config.jiraClient.NewRequest("GET", apiEndpoint, nil)
		search := new(RawSearch)
		_, err := config.jiraClient.Do(req, &search)

		if err == nil {
			users := make([]RawUser, 0)
			for _, v := range search.Users.Users {
				if !strings.HasPrefix(v.AccountId, "qm:") {
					users = append(users, v)
				}
			}
			total := len(users)

			if total == 1 {
				d.SetId(users[0].AccountId)
				d.Set("account_id", users[0].AccountId)
				d.Set("display_name", users[0].DisplayName)
				d.Set("email", id)
			} else {
				names := make([]string, 0)
				for _, v := range users {
					names = append(names, fmt.Sprintf("%s/%s", v.AccountId, v.DisplayName))
				}
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary: fmt.Sprintf("no exact match for %s, found %d users (%s)",
						id, total, strings.Join(names, ",")),
				})
				return diags
			}
		} else {
			return diag.FromErr(err)
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

	apiEndpoint := fmt.Sprintf("/rest/api/3/group/user?groupname=%s&accountId=%s", url.QueryEscape(groupname), username)
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
		apiEndpoint := fmt.Sprintf("/users/%s/manage/lifecycle/", id)
		if active {
			apiEndpoint += "enable"
		} else {
			apiEndpoint += "disable"
		}

		req, err := config.adminClient.NewRequestWithContext(
			ctx,
			"POST",
			apiEndpoint,
			map[string]interface{}{"message": "managed by terraform"})

		if err != nil {
			return diag.FromErr(err)
		}

		_, err2 := config.adminClient.Do(req, nil)

		if err2 != nil {
			return diag.FromErr(err2)
		}
	}

	if email := d.Get("email"); d.HasChange("email") {
		apiEndpoint := fmt.Sprintf("/users/%s/manage/email", id)

		req, err := config.adminClient.NewRequestWithContext(
			ctx,
			"PUT",
			apiEndpoint,
			map[string]interface{}{"email": email})

		if err != nil {
			return diag.FromErr(err)
		}

		_, err2 := config.adminClient.Do(req, nil)

		if err2 != nil {
			return diag.FromErr(err2)
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
