package contentful

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulAPIKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: wrapApiKey(resourceCreateAPIKey),
		ReadContext:   wrapApiKey(resourceReadAPIKey),
		UpdateContext: wrapApiKey(resourceUpdateAPIKey),
		DeleteContext: wrapApiKey(resourceDeleteAPIKey),

		Schema: map[string]*schema.Schema{
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"access_token": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"space_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

type ContentfulAPIKeyClient interface {
	Get(context.Context, string, string) (*contentful.APIKey, error)
	Upsert(context.Context, string, *contentful.APIKey) error
	Delete(context.Context, string, *contentful.APIKey) error
}

func wrapApiKey(f func(ctx context.Context, d *schema.ResourceData, apiKey ContentfulAPIKeyClient) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
		client := m.(*contentful.Client)
		return f(ctx, d, client.APIKeys)
	}
}

func resourceCreateAPIKey(ctx context.Context, d *schema.ResourceData, client ContentfulAPIKeyClient) (diags diag.Diagnostics) {
	apiKey := &contentful.APIKey{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	err := client.Upsert(ctx, d.Get("space_id").(string), apiKey)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err := setAPIKeyProperties(d, apiKey); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(apiKey.Sys.ID)

	return nil
}

func resourceUpdateAPIKey(ctx context.Context, d *schema.ResourceData, client ContentfulAPIKeyClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	apiKeyID := d.Id()

	apiKey, err := client.Get(ctx, spaceID, apiKeyID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	apiKey.Name = d.Get("name").(string)
	apiKey.Description = d.Get("description").(string)

	err = client.Upsert(ctx, spaceID, apiKey)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err := setAPIKeyProperties(d, apiKey); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(apiKey.Sys.ID)

	return nil
}

func resourceReadAPIKey(ctx context.Context, d *schema.ResourceData, client ContentfulAPIKeyClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	apiKeyID := d.Id()

	apiKey, err := client.Get(ctx, spaceID, apiKeyID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}

	err = setAPIKeyProperties(d, apiKey)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func resourceDeleteAPIKey(ctx context.Context, d *schema.ResourceData, client ContentfulAPIKeyClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	apiKeyID := d.Id()

	apiKey, err := client.Get(ctx, spaceID, apiKeyID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Delete(ctx, spaceID, apiKey)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func setAPIKeyProperties(d *schema.ResourceData, apiKey *contentful.APIKey) error {
	if err := d.Set("space_id", apiKey.Sys.Space.Sys.ID); err != nil {
		return err
	}

	if err := d.Set("version", apiKey.Sys.Version); err != nil {
		return err
	}

	if err := d.Set("name", apiKey.Name); err != nil {
		return err
	}

	if err := d.Set("description", apiKey.Description); err != nil {
		return err
	}

	if err := d.Set("access_token", apiKey.AccessToken); err != nil {
		return err
	}

	return nil
}
