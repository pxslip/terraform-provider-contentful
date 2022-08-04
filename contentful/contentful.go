package contentful

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	contentful "github.com/kitagry/contentful-go"
)

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
