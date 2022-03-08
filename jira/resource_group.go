package jira

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GroupRequest The struct sent to the JIRA instance to create a new Group
type GroupRequest struct {
	Name string `json:"name,omitempty" structs:"name,omitempty"`
}

// resourceGroup is used to define a JIRA issue
func resourceGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGroupCreate,
		ReadContext:   resourceGroupRead,
		DeleteContext: resourceGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

// resourceGroupCreate creates a new jira issue using the jira api
func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	group := new(GroupRequest)
	group.Name = d.Get("name").(string)

	err := request(config.jiraClient, "POST", groupAPIEndpoint, group, nil)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Request failed: %s", err.Error()),
		})
		return diags
	}

	d.SetId(group.Name)

	return resourceGroupRead(ctx, d, m)
}

// resourceGroupRead reads issue details using jira api
func resourceGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	_, _, err := config.jiraClient.Group.Get(d.Id())
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("getting jira group failed: %s", err.Error()),
		})
		return diags
	}

	d.Set("name", d.Id())

	return nil
}

// resourceGroupDelete deletes jira issue using the jira api
func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	config := m.(*Config)

	relativeURL, _ := url.Parse(groupAPIEndpoint)

	query := relativeURL.Query()
	query.Set("groupname", d.Get("name").(string))

	relativeURL.RawQuery = query.Encode()

	err := request(config.jiraClient, "DELETE", relativeURL.String(), nil, nil)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Request failed: %s", err.Error()),
		})
		return diags
	}

	return nil
}
