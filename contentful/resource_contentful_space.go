package contentful

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulSpace() *schema.Resource {
	return &schema.Resource{
		CreateContext: wrapSpace(resourceSpaceCreate),
		ReadContext:   wrapSpace(resourceSpaceRead),
		UpdateContext: wrapSpace(resourceSpaceUpdate),
		DeleteContext: wrapSpace(resourceSpaceDelete),

		Schema: map[string]*schema.Schema{
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			// Space specific props
			"default_locale": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "en",
			},
		},
	}
}

func wrapSpace(f func(ctx context.Context, d *schema.ResourceData, client ContentfulSpaceClient) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
		client := m.(*contentful.Client)
		return f(ctx, d, client.Spaces)
	}
}

func resourceSpaceCreate(ctx context.Context, d *schema.ResourceData, client ContentfulSpaceClient) (diags diag.Diagnostics) {
	space := &contentful.Space{
		Name:          d.Get("name").(string),
		DefaultLocale: d.Get("default_locale").(string),
	}

	err := client.Upsert(ctx, space)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = updateSpaceProperties(d, space)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(space.Sys.ID)

	return nil
}

func resourceSpaceRead(ctx context.Context, d *schema.ResourceData, client ContentfulSpaceClient) (diags diag.Diagnostics) {
	spaceID := d.Id()

	_, err := client.Get(ctx, spaceID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func resourceSpaceUpdate(ctx context.Context, d *schema.ResourceData, client ContentfulSpaceClient) (diags diag.Diagnostics) {
	spaceID := d.Id()
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	space, err := client.Get(ctx, spaceID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	space.Name = d.Get("name").(string)

	err = client.Upsert(ctx, space)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = updateSpaceProperties(d, space)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func resourceSpaceDelete(ctx context.Context, d *schema.ResourceData, client ContentfulSpaceClient) (diags diag.Diagnostics) {
	spaceID := d.Id()

	space, err := client.Get(ctx, spaceID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Delete(ctx, space)
	if _, ok := err.(contentful.NotFoundError); ok {
		return nil
	}
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func updateSpaceProperties(d *schema.ResourceData, space *contentful.Space) error {
	err := d.Set("version", space.Sys.Version)
	if err != nil {
		return err
	}

	err = d.Set("name", space.Name)
	if err != nil {
		return err
	}

	return nil
}
