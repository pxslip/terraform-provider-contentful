package contentful

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulAsset() *schema.Resource {
	return &schema.Resource{
		CreateContext: wrapAsset(resourceCreateAsset),
		ReadContext:   wrapAsset(resourceReadAsset),
		UpdateContext: wrapAsset(resourceUpdateAsset),
		DeleteContext: wrapAsset(resourceDeleteAsset),

		Schema: map[string]*schema.Schema{
			"asset_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"locale": {
				Type:     schema.TypeString,
				Required: true,
			},
			"space_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"fields": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content": {
										Type:     schema.TypeString,
										Required: true,
									},
									"locale": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"description": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content": {
										Type:     schema.TypeString,
										Required: true,
									},
									"locale": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"file": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"upload": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"details": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"size": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"image": {
													Type:     schema.TypeSet,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"width": {
																Type:     schema.TypeInt,
																Required: true,
															},
															"height": {
																Type:     schema.TypeInt,
																Required: true,
															},
														},
													},
												},
											},
										},
									},
									"file_name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"content_type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"published": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"archived": {
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

type ContentfulAssetClient interface {
	Get(ctx context.Context, spaceID string, assetID string) (*contentful.Asset, error)
	Upsert(ctx context.Context, spaceID string, asset *contentful.Asset) error
	Process(ctx context.Context, spaceID string, asset *contentful.Asset) error
	Delete(ctx context.Context, spaceID string, asset *contentful.Asset) error
	Publish(ctx context.Context, spaceID string, asset *contentful.Asset) error
	Unpublish(ctx context.Context, spaceID string, asset *contentful.Asset) error
	Archive(ctx context.Context, spaceID string, asset *contentful.Asset) error
	Unarchive(ctx context.Context, spaceID string, asset *contentful.Asset) error
}

func wrapAsset(f func(ctx context.Context, d *schema.ResourceData, client ContentfulAssetClient) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
		client := m.(*contentful.Client)
		return f(ctx, d, client.Assets)
	}
}

func resourceCreateAsset(ctx context.Context, d *schema.ResourceData, client ContentfulAssetClient) (diags diag.Diagnostics) {
	fields := d.Get("fields").([]interface{})[0].(map[string]interface{})

	localizedTitle := map[string]string{}
	rawTitle := fields["title"].([]interface{})
	for i := 0; i < len(rawTitle); i++ {
		field := rawTitle[i].(map[string]interface{})
		localizedTitle[field["locale"].(string)] = field["content"].(string)
	}

	localizedDescription := map[string]string{}
	rawDescription := fields["description"].([]interface{})
	for i := 0; i < len(rawDescription); i++ {
		field := rawDescription[i].(map[string]interface{})
		localizedDescription[field["locale"].(string)] = field["content"].(string)
	}

	files := fields["file"].(*schema.Set).List()
	if len(files) != 1 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "file should be single item",
		})
		return
	}
	file := files[0].(map[string]interface{})

	asset := &contentful.Asset{
		Sys: &contentful.Sys{
			ID:      d.Get("asset_id").(string),
			Version: 0,
		},
		Locale: d.Get("locale").(string),
		Fields: &contentful.AssetFields{
			Title:       localizedTitle,
			Description: localizedDescription,
			File: map[string]*contentful.File{
				d.Get("locale").(string): {
					FileName:    file["file_name"].(string),
					ContentType: file["content_type"].(string),
				},
			},
		},
	}

	if url, ok := file["url"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].URL = url
	}

	if upload, ok := file["upload"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].UploadURL = upload
	}

	if details, ok := file["details"].(*contentful.FileDetails); ok {
		asset.Fields.File[d.Get("locale").(string)].Details = details
	}

	err := client.Upsert(ctx, d.Get("space_id").(string), asset)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Process(ctx, d.Get("space_id").(string), asset)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(asset.Sys.ID)

	if err := setAssetProperties(d, asset); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setAssetState(ctx, d, client)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func resourceUpdateAsset(ctx context.Context, d *schema.ResourceData, client ContentfulAssetClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	asset, err := client.Get(ctx, spaceID, assetID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	fields := d.Get("fields").([]interface{})[0].(map[string]interface{})

	localizedTitle := map[string]string{}
	rawTitle := fields["title"].([]interface{})
	for i := 0; i < len(rawTitle); i++ {
		field := rawTitle[i].(map[string]interface{})
		localizedTitle[field["locale"].(string)] = field["content"].(string)
	}

	localizedDescription := map[string]string{}
	rawDescription := fields["description"].([]interface{})
	for i := 0; i < len(rawDescription); i++ {
		field := rawDescription[i].(map[string]interface{})
		localizedDescription[field["locale"].(string)] = field["content"].(string)
	}

	files := fields["file"].(*schema.Set).List()
	if len(files) != 1 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "file should be single item",
		})
		return
	}
	file := files[0].(map[string]interface{})

	asset = &contentful.Asset{
		Sys: &contentful.Sys{
			ID:      d.Get("asset_id").(string),
			Version: d.Get("version").(int),
		},
		Locale: d.Get("locale").(string),
		Fields: &contentful.AssetFields{
			Title:       localizedTitle,
			Description: localizedDescription,
			File: map[string]*contentful.File{
				d.Get("locale").(string): {
					FileName:    file["file_name"].(string),
					ContentType: file["content_type"].(string),
				},
			},
		},
	}

	if url, ok := file["url"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].URL = url
	}

	if upload, ok := file["upload"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].UploadURL = upload
	}

	if details, ok := file["file_details"].(*contentful.FileDetails); ok {
		asset.Fields.File[d.Get("locale").(string)].Details = details
	}

	err = client.Upsert(ctx, d.Get("space_id").(string), asset)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Process(ctx, d.Get("space_id").(string), asset)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(asset.Sys.ID)

	if err := setAssetProperties(d, asset); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setAssetState(ctx, d, client)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func setAssetState(ctx context.Context, d *schema.ResourceData, client ContentfulAssetClient) (err error) {
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()

	asset, _ := client.Get(ctx, spaceID, assetID)

	if d.Get("published").(bool) && asset.Sys.PublishedAt == "" {
		err = client.Publish(ctx, spaceID, asset)
	} else if !d.Get("published").(bool) && asset.Sys.PublishedAt != "" {
		err = client.Unpublish(ctx, spaceID, asset)
	}

	if d.Get("archived").(bool) && asset.Sys.ArchivedAt == "" {
		err = client.Archive(ctx, spaceID, asset)
	} else if !d.Get("archived").(bool) && asset.Sys.ArchivedAt != "" {
		err = client.Unarchive(ctx, spaceID, asset)
	}

	err = setAssetProperties(d, asset)

	return err
}

func resourceReadAsset(ctx context.Context, d *schema.ResourceData, client ContentfulAssetClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()

	asset, err := client.Get(ctx, spaceID, assetID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setAssetProperties(d, asset)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func resourceDeleteAsset(ctx context.Context, d *schema.ResourceData, client ContentfulAssetClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()

	asset, err := client.Get(ctx, spaceID, assetID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Delete(ctx, spaceID, asset)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func setAssetProperties(d *schema.ResourceData, asset *contentful.Asset) (err error) {
	if err = d.Set("space_id", asset.Sys.Space.Sys.ID); err != nil {
		return err
	}

	if err = d.Set("version", asset.Sys.Version); err != nil {
		return err
	}

	return err
}
