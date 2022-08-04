package contentful

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulLocale() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCreateLocale,
		ReadContext:   resourceReadLocale,
		UpdateContext: resourceUpdateLocale,
		DeleteContext: resourceDeleteLocale,

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
			"code": {
				Type:     schema.TypeString,
				Required: true,
			},
			"fallback_code": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "en-US",
			},
			"optional": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cda": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"cma": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceCreateLocale(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)

	locale := &contentful.Locale{
		Name:         d.Get("name").(string),
		Code:         d.Get("code").(string),
		FallbackCode: d.Get("fallback_code").(string),
		Optional:     d.Get("optional").(bool),
		CDA:          d.Get("cda").(bool),
		CMA:          d.Get("cma").(bool),
	}

	err := client.Locales.Upsert(ctx, spaceID, locale)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	err = setLocaleProperties(d, locale)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	d.SetId(locale.Sys.ID)

	return nil
}

func resourceReadLocale(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	localeID := d.Id()

	locale, err := client.Locales.Get(ctx, spaceID, localeID)
	if _, ok := err.(*contentful.NotFoundError); ok {
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

	err = setLocaleProperties(d, locale)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}
	return
}

func resourceUpdateLocale(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	localeID := d.Id()
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	locale, err := client.Locales.Get(ctx, spaceID, localeID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	locale.Name = d.Get("name").(string)
	locale.Code = d.Get("code").(string)
	locale.FallbackCode = d.Get("fallback_code").(string)
	locale.Optional = d.Get("optional").(bool)
	locale.CDA = d.Get("cda").(bool)
	locale.CMA = d.Get("cma").(bool)

	err = client.Locales.Upsert(ctx, spaceID, locale)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	err = setLocaleProperties(d, locale)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	return
}

func resourceDeleteLocale(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	localeID := d.Id()

	locale, err := client.Locales.Get(ctx, spaceID, localeID)
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	err = client.Locales.Delete(ctx, spaceID, locale)
	if _, ok := err.(*contentful.NotFoundError); ok {
		return nil
	}

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
		})
		return
	}

	return nil
}

func setLocaleProperties(d *schema.ResourceData, locale *contentful.Locale) error {
	err := d.Set("name", locale.Name)
	if err != nil {
		return err
	}

	err = d.Set("code", locale.Code)
	if err != nil {
		return err
	}

	err = d.Set("fallback_code", locale.FallbackCode)
	if err != nil {
		return err
	}

	err = d.Set("optional", locale.Optional)
	if err != nil {
		return err
	}

	err = d.Set("cda", locale.CDA)
	if err != nil {
		return err
	}

	err = d.Set("cma", locale.CMA)
	if err != nil {
		return err
	}

	return nil
}
