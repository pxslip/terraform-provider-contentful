package contentful

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	contentful "github.com/kitagry/contentful-go"
)

func TestContentfulErrorToDiagnostic(t *testing.T) {
	tests := map[string]struct {
		err    error
		expect diag.Diagnostics
	}{
		"ErrorResponse should return diagnostics": {
			err: contentful.ErrorResponse{
				Message: "msg",
				Details: &contentful.ErrorDetails{
					Errors: []*contentful.ErrorDetail{
						{
							Details: "details",
						},
					},
				},
			},
			expect: diag.Diagnostics{
				{
					Summary: "msg",
					Detail:  "details",
				},
			},
		},
		"ErrorResponse should return each diagnostics": {
			err: contentful.ErrorResponse{
				Message: "msg",
				Details: &contentful.ErrorDetails{
					Errors: []*contentful.ErrorDetail{
						{
							Details: "details1",
							Path: []interface{}{
								"field",
								1.,
								"entry",
							},
						},
						{
							Details: "details2",
						},
					},
				},
			},
			expect: diag.Diagnostics{
				{
					Summary: "msg",
					Detail:  "details1",
					AttributePath: cty.Path{
						cty.GetAttrStep{Name: "field"},
						cty.IndexStep{Key: cty.NumberIntVal(1)},
						cty.GetAttrStep{Name: "entry"},
					},
				},
				{
					Summary: "msg",
					Detail:  "details2",
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := contentfulErrorToDiagnostic(tt.err)
			if diff := cmp.Diff(tt.expect, got, cmp.AllowUnexported(cty.IndexStep{}, cty.GetAttrStep{}), cmpopts.IgnoreFields(cty.Value{}, "ty", "v")); diff != "" {
				t.Errorf("contentfulErrorToDiagnostic result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
