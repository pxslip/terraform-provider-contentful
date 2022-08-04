package contentful

import (
	"context"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	contentful "github.com/kitagry/contentful-go"
)

type ContentfulAPIKeyClient interface {
	Get(context.Context, string, string) (*contentful.APIKey, error)
	Upsert(context.Context, string, *contentful.APIKey) error
	Delete(context.Context, string, *contentful.APIKey) error
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

type ContentfulContentTypeClient interface {
	Get(ctx context.Context, env *contentful.Environment, contentTypeID string) (*contentful.ContentType, error)
	Upsert(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
	Activate(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
	Deactivate(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
	Delete(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
}

type ContentfulEntryClient interface {
	Get(ctx context.Context, env *contentful.Environment, entryID string) (*contentful.Entry, error)
	Upsert(ctx context.Context, env *contentful.Environment, contentTypeID string, e *contentful.Entry) error
	Delete(ctx context.Context, env *contentful.Environment, entryID string) error

	Publish(ctx context.Context, env *contentful.Environment, entry *contentful.Entry) error
	Unpublish(ctx context.Context, env *contentful.Environment, entry *contentful.Entry) error
	Archive(ctx context.Context, env *contentful.Environment, entry *contentful.Entry) error
	Unarchive(ctx context.Context, env *contentful.Environment, entry *contentful.Entry) error
}

type ContentfulEnvironmentClient interface {
	Get(ctx context.Context, spaceID string, environmentID string) (*contentful.Environment, error)
	Upsert(ctx context.Context, spaceID string, e *contentful.Environment) error
	Delete(ctx context.Context, spaceID string, e *contentful.Environment) error
}

type ContentfulLocaleClient interface {
	Get(context.Context, string, string) (*contentful.Locale, error)
	Upsert(context.Context, string, *contentful.Locale) error
	Delete(context.Context, string, *contentful.Locale) error
}

type ContentfulSpaceClient interface {
	Get(context.Context, string) (*contentful.Space, error)
	Upsert(context.Context, *contentful.Space) error
	Delete(context.Context, *contentful.Space) error
}

type ContentfulWebhookClient interface {
	Get(context.Context, string, string) (*contentful.Webhook, error)
	Upsert(context.Context, string, *contentful.Webhook) error
	Delete(context.Context, string, *contentful.Webhook) error
}

func contentfulErrorToDiagnostic(err error) diag.Diagnostics {
	switch v := err.(type) {
	case contentful.ErrorResponse:
		return convertContentfulErrorResponse(&v)
	case contentful.ValidationFailedError:
		res, ok := v.ErrorResponse()
		if ok {
			return convertContentfulErrorResponse(res)
		}
	}
	return diag.Diagnostics{
		{
			Severity: diag.Error,
			Summary:  err.Error(),
		},
	}
}

func convertContentfulErrorResponse(v *contentful.ErrorResponse) diag.Diagnostics {
	diags := make(diag.Diagnostics, 0)
	for _, e := range v.Details.Errors {
		var path cty.Path
		pathInterface, ok := e.Path.([]interface{})
		if ok {
			for _, p := range pathInterface {
				switch v := p.(type) {
				case string:
					path = append(path, cty.GetAttrStep{Name: v})
				case float64:
					path = append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(v))})
				case int:
					path = append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(v))})
				case int64:
					path = append(path, cty.IndexStep{Key: cty.NumberIntVal(v)})
				}
			}
		}
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       v.Message,
			Detail:        e.Details,
			AttributePath: path,
		})
	}
	return diags
}
