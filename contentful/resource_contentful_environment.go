package contentful

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulEnvironment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCreateEnvironment,
		ReadContext:   resourceReadEnvironment,
		UpdateContext: resourceUpdateEnvironment,
		DeleteContext: resourceDeleteEnvironment,

		Schema: map[string]*schema.Schema{
			"version": {
				Type:     schema.TypeInt,
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
		},
	}
}

func resourceCreateEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)

	environment := &contentful.Environment{
		Name: d.Get("name").(string),
	}

	err := client.Environments.Upsert(ctx, d.Get("space_id").(string), environment)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	if err := setEnvironmentProperties(d, environment); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	d.SetId(environment.Name)

	return nil
}

func resourceUpdateEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	environmentID := d.Id()
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	environment, err := client.Environments.Get(ctx, spaceID, environmentID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	environment.Name = d.Get("name").(string)

	err = client.Environments.Upsert(ctx, spaceID, environment)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	if err := setEnvironmentProperties(d, environment); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	d.SetId(environment.Sys.ID)

	return nil
}

func resourceReadEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	environmentID := d.Id()

	environment, err := client.Environments.Get(ctx, spaceID, environmentID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	err = setEnvironmentProperties(d, environment)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}
	return
}

func resourceDeleteEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	environmentID := d.Id()

	environment, err := client.Environments.Get(ctx, spaceID, environmentID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	err = client.Environments.Delete(ctx, spaceID, environment)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}
	return
}

func setEnvironmentProperties(d *schema.ResourceData, environment *contentful.Environment) error {
	if err := d.Set("space_id", environment.Sys.Space.Sys.ID); err != nil {
		return err
	}

	if err := d.Set("version", environment.Sys.Version); err != nil {
		return err
	}

	if err := d.Set("name", environment.Name); err != nil {
		return err
	}

	return nil
}
