package contentful

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	contentful "github.com/kitagry/contentful-go"
)

func TestNewField(t *testing.T) {
	tests := map[string]struct {
		newField map[string]interface{}
		i        int

		expectField *contentful.Field
		expectDiags diag.Diagnostics
	}{
		"correct field": {
			newField: map[string]interface{}{
				"id":        "id",
				"name":      "name",
				"type":      "type",
				"localized": true,
				"required":  true,
				"disabled":  false,
				"omitted":   false,
				"items":     []interface{}(nil),
			},
			expectField: &contentful.Field{
				ID:        "id",
				Name:      "name",
				Type:      "type",
				Localized: true,
				Required:  true,
				Disabled:  false,
				Omitted:   false,
			},
		},
		"invalid json": {
			newField: map[string]interface{}{
				"id":          "id",
				"name":        "name",
				"type":        "type",
				"localized":   true,
				"required":    true,
				"disabled":    false,
				"omitted":     false,
				"validations": []interface{}{"invalid json"},
				"items":       []interface{}(nil),
			},
			i: 0,
			expectDiags: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "validation format is invalid.",
					Detail:   "invalid character 'i' looking for beginning of value",
					AttributePath: cty.Path{
						cty.GetAttrStep{Name: "field"},
						cty.IndexStep{Key: cty.NumberIntVal(0)},
						cty.GetAttrStep{Name: "validations"},
					},
				},
			},
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			gotField, gotDiags := newField(tt.newField, tt.i)
			if diff := cmp.Diff(tt.expectField, gotField); diff != "" {
				t.Errorf("gotField result diff (-expect, +got)\n%s", diff)
			}
			if diff := cmp.Diff(tt.expectDiags, gotDiags, cmp.AllowUnexported(cty.IndexStep{}, cty.GetAttrStep{}), cmpopts.IgnoreFields(cty.Value{}, "ty", "v")); diff != "" {
				t.Errorf("gotDiags result diff (-expect, +got)\n%s", diff)
			}
		})
	}
}
