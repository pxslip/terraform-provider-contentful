package contentful

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulEnvironment() *schema.Resource {
	return &schema.Resource{
		CreateContext: wrapEnvironment(resourceCreateEnvironment),
		ReadContext:   wrapEnvironment(resourceReadEnvironment),
		UpdateContext: wrapEnvironment(resourceUpdateEnvironment),
		DeleteContext: wrapEnvironment(resourceDeleteEnvironment),

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

type ContentfulEnvironmentClient interface {
	Get(ctx context.Context, spaceID string, environmentID string) (*contentful.Environment, error)
	Upsert(ctx context.Context, spaceID string, e *contentful.Environment) error
	Delete(ctx context.Context, spaceID string, e *contentful.Environment) error
}

func wrapEnvironment(f func(ctx context.Context, d *schema.ResourceData, apiKey ContentfulEnvironmentClient) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
		client := m.(*contentful.Client)
		return f(ctx, d, client.Environments)
	}
}

func resourceCreateEnvironment(ctx context.Context, d *schema.ResourceData, client ContentfulEnvironmentClient) (diags diag.Diagnostics) {
	environment := &contentful.Environment{
		Name: d.Get("name").(string),
	}

	err := client.Upsert(ctx, d.Get("space_id").(string), environment)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err := setEnvironmentProperties(d, environment); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(environment.Name)

	return nil
}

func resourceUpdateEnvironment(ctx context.Context, d *schema.ResourceData, client ContentfulEnvironmentClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	environmentID := d.Id()
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	environment, err := client.Get(ctx, spaceID, environmentID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	environment.Name = d.Get("name").(string)

	err = client.Upsert(ctx, spaceID, environment)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err := setEnvironmentProperties(d, environment); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(environment.Sys.ID)

	return nil
}

func resourceReadEnvironment(ctx context.Context, d *schema.ResourceData, client ContentfulEnvironmentClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	environmentID := d.Id()

	environment, err := client.Get(ctx, spaceID, environmentID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setEnvironmentProperties(d, environment)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func resourceDeleteEnvironment(ctx context.Context, d *schema.ResourceData, client ContentfulEnvironmentClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	environmentID := d.Id()

	environment, err := client.Get(ctx, spaceID, environmentID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Delete(ctx, spaceID, environment)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
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
