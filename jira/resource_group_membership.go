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

// GroupMembership The struct sent to the JIRA instance to create a new GroupMembership
type GroupMembership struct {
	AccountId string `json:"accountId,omitempty" structs:"accountId,omitempty"`
}

// Groups List of groups the user belongs to
type Group struct {
	Name string `json:"name,omitempty" structs:"name,omitempty"`
}

type Groups struct {
	Items []Group `json:"items,omitempty" structs:"items,omitempty"`
}

// UserGroups Wrapper for the groups of a user
type UserGroups struct {
	Groups Groups `json:"groups,omitempty" structs:"groups,omitempty"`
}

func getGroups(jiraClient *jira.Client, accountId string) (*UserGroups, *jira.Response, error) {
	relativeURL, _ := url.Parse("/rest/api/3/user")
	query := relativeURL.Query()
	query.Set("accountId", accountId)
	query.Set("expand", "groups")

	relativeURL.RawQuery = query.Encode()

	req, err := jiraClient.NewRequest("GET", relativeURL.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	user := new(UserGroups)
	resp, err := jiraClient.Do(req, user)
	if err != nil {
		return nil, resp, jira.NewJiraError(resp, err)
	}
	return user, resp, nil
}

// resourceGroupMembership is used to define a JIRA group membership
func resourceGroupMembership() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGroupMembershipCreate,
		ReadContext:   resourceGroupMembershipRead,
		DeleteContext: resourceGroupMembershipDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"account_id": {
				Description: "The Atlassian account id.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"group": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

// resourceGroupMembershipCreate creates a new jira group membership using the jira api
func resourceGroupMembershipCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	accountId := d.Get("account_id").(string)
	group := d.Get("group").(string)

	groupMembership := new(GroupMembership)
	groupMembership.AccountId = accountId

	relativeURL, _ := url.Parse(groupUserAPIEndpoint)
	query := relativeURL.Query()
	query.Set("groupname", group)
	relativeURL.RawQuery = query.Encode()

	err := request(config.jiraClient, "POST", relativeURL.String(), groupMembership, nil)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Request failed: %s", err.Error()),
		})
		return diags
	}

	d.SetId(fmt.Sprintf("%s/%s", accountId, group))

	return resourceGroupMembershipRead(ctx, d, m)
}

// resourceGroupMembershipRead reads issue details using jira api
func resourceGroupMembershipRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	components := strings.SplitN(d.Id(), "/", 2)
	accountId := components[0]
	groupname := components[1]

	groups, _, err := getGroups(config.jiraClient, accountId)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Request failed: %s", err.Error()),
		})
		return diags
	}

	d.Set("accountId", accountId)
	d.Set("group", groupname)

	for _, group := range groups.Groups.Items {
		if group.Name == groupname {
			return nil
		}
	}

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("Cannot find group: %s", groupname),
	})
	return diags
}

// resourceGroupMembershipDelete deletes jira issue using the jira api
func resourceGroupMembershipDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	relativeURL, _ := url.Parse(groupUserAPIEndpoint)

	query := relativeURL.Query()
	query.Set("accountId", d.Get("accountId").(string))
	query.Set("groupname", d.Get("group").(string))

	relativeURL.RawQuery = query.Encode()

	client := config.jiraClient
	req, err := client.NewRequest("DELETE", relativeURL.String(), nil)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Cannot build request: %s", err.Error()),
		})
		return diags
	}

	_, err = client.Do(req, nil)

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Request failed: %s", err.Error()),
		})
		return diags
	}

	return nil
}
