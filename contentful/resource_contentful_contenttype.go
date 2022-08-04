package contentful

import (
	"context"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulContentType() *schema.Resource {
	return &schema.Resource{
		CreateContext: wrapContentType(resourceContentTypeCreate),
		ReadContext:   wrapContentType(resourceContentTypeRead),
		UpdateContext: wrapContentType(resourceContentTypeUpdate),
		DeleteContext: wrapContentType(resourceContentTypeDelete),

		Schema: map[string]*schema.Schema{
			"space_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"display_field": {
				Type:     schema.TypeString,
				Required: true,
			},
			"content_type_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"env_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"field": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"link_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"items": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"link_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"validations": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"required": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"localized": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"disabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"omitted": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"validations": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

type ContentfulContentTypeClient interface {
	Get(ctx context.Context, env *contentful.Environment, contentTypeID string) (*contentful.ContentType, error)
	Upsert(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
	Activate(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
	Deactivate(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
	Delete(ctx context.Context, env *contentful.Environment, ct *contentful.ContentType) error
}

func wrapContentType(f func(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, apiKey ContentfulContentTypeClient) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
		client := m.(*contentful.Client)
		spaceID := d.Get("space_id").(string)
		envID := d.Get("env_id").(string)
		env, err := client.Environments.Get(ctx, spaceID, envID)
		if err != nil {
			diags = append(diags, contentfulErrorToDiagnostic(err)...)
			return
		}
		return f(ctx, d, env, client.ContentTypes)
	}
}

func resourceContentTypeCreate(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulContentTypeClient) (diags diag.Diagnostics) {
	ct := &contentful.ContentType{
		Name:         d.Get("name").(string),
		DisplayField: d.Get("display_field").(string),
		Fields:       []*contentful.Field{},
		Sys: &contentful.Sys{
			ID: d.Get("content_type_id").(string),
		},
	}

	id := d.Get("content_type_id")

	if id != nil {
		ct.Sys = &contentful.Sys{
			ID: id.(string),
		}
	}

	if description, ok := d.GetOk("description"); ok {
		ct.Description = description.(string)
	}

	rawField := d.Get("field").([]interface{})
	for i := 0; i < len(rawField); i++ {
		field := rawField[i].(map[string]interface{})

		contentfulField := &contentful.Field{
			ID:        field["id"].(string),
			Name:      field["name"].(string),
			Type:      field["type"].(string),
			Localized: field["localized"].(bool),
			Required:  field["required"].(bool),
			Disabled:  field["disabled"].(bool),
			Omitted:   field["omitted"].(bool),
		}

		if linkType, ok := field["link_type"].(string); ok {
			contentfulField.LinkType = linkType
		}

		if validations, ok := field["validations"].([]interface{}); ok {
			parsedValidations, err := contentful.ParseValidations(validations)
			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "validation format is invalid.",
					Detail:   err.Error(),
					AttributePath: cty.Path{
						cty.GetAttrStep{Name: "field"},
						cty.IndexStep{Key: cty.NumberIntVal(int64(i))},
						cty.GetAttrStep{Name: "validations"},
					},
				})
				continue
			}

			contentfulField.Validations = parsedValidations
		}

		if items := processItems(field["items"].([]interface{})); items != nil {
			contentfulField.Items = items
		}

		ct.Fields = append(ct.Fields, contentfulField)
	}

	if diags.HasError() {
		return
	}

	if err := upsertAndActivate(ctx, client, env, ct); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err := setContentTypeProperties(d, ct); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(ct.Sys.ID)

	return nil
}

func resourceContentTypeRead(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulContentTypeClient) (diags diag.Diagnostics) {
	_, err := client.Get(ctx, env, d.Id())
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func resourceContentTypeUpdate(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulContentTypeClient) (diags diag.Diagnostics) {
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	ct, err := client.Get(ctx, env, d.Id())
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	ct.Name = d.Get("name").(string)
	ct.DisplayField = d.Get("display_field").(string)

	if description, ok := d.GetOk("description"); ok {
		ct.Description = description.(string)
	}

	if d.HasChange("field") {
		old, nw := d.GetChange("field")

		firstApplyFields, secondApplyFields, shouldSecondApply := checkFieldsToOmit(old.([]interface{}), nw.([]interface{}))

		ct.Fields = firstApplyFields
		// To remove a field from a content type 4 API calls need to be made.
		// Omit the removed fields and publish the new version of the content type,
		// followed by the field removal and final publish.
		if err = upsertAndActivate(ctx, client, env, ct); err != nil {
			diags = append(diags, contentfulErrorToDiagnostic(err)...)
			return
		}

		if shouldSecondApply {
			ct.Fields = secondApplyFields
			if err = upsertAndActivate(ctx, client, env, ct); err != nil {
				diags = append(diags, contentfulErrorToDiagnostic(err)...)
				return
			}
		}
	}

	ct.Fields = newFields(d.Get("field").([]interface{}))
	if err = upsertAndActivate(ctx, client, env, ct); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setContentTypeProperties(d, ct)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func upsertAndActivate(ctx context.Context, client ContentfulContentTypeClient, env *contentful.Environment, ct *contentful.ContentType) error {
	if err := client.Upsert(ctx, env, ct); err != nil {
		return err
	}

	if err := client.Activate(ctx, env, ct); err != nil {
		return err
	}
	return nil
}

func resourceContentTypeDelete(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulContentTypeClient) (diags diag.Diagnostics) {
	ct, err := client.Get(ctx, env, d.Id())
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err = client.Deactivate(ctx, env, ct); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err = client.Delete(ctx, env, ct); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func setContentTypeProperties(d *schema.ResourceData, ct *contentful.ContentType) (err error) {
	if err = d.Set("version", ct.Sys.Version); err != nil {
		return err
	}

	return nil
}

// Contentful API should omit the field.
// And if user want to change field type, user should delete the field completely before user create new field type field.
func checkFieldsToOmit(oldFields, newFields []interface{}) (firstApplyFields, secondApplyFields []*contentful.Field, shouldSecondApply bool) {
	getFieldFromID := func(fields []interface{}, id string) (map[string]interface{}, bool) {
		for _, field := range fields {
			castedField := field.(map[string]interface{})
			if castedField["id"].(string) == id {
				return castedField, true
			}
		}
		return nil, false
	}

	for i := 0; i < len(oldFields); i++ {
		oldFieldMap := oldFields[i].(map[string]interface{})

		newFieldMap, ok := getFieldFromID(newFields, oldFieldMap["id"].(string))

		toOmitted := false
		if !ok {
			// field was deleted
			toOmitted = true
		} else {
			if oldFieldMap["type"].(string) != newFieldMap["type"].(string) {
				toOmitted = true
			}
		}

		shouldDelete := false
		if ok {
			// if field type is changed, should delete field completely
			if oldFieldMap["type"].(string) != newFieldMap["type"].(string) {
				shouldDelete = true
			}
		}

		field := newField(oldFieldMap)
		if toOmitted {
			field.Omitted = true
		}

		firstApplyFields = append(firstApplyFields, field)
		if !shouldDelete {
			secondApplyFields = append(secondApplyFields, field)
		} else {
			shouldSecondApply = true
		}
	}
	return
}

func newFields(newFields []interface{}) []*contentful.Field {
	result := make([]*contentful.Field, len(newFields))
	for i := 0; i < len(newFields); i++ {
		newFieldMap := newFields[i].(map[string]interface{})
		result[i] = newField(newFieldMap)
	}
	return result
}

func newField(newField map[string]interface{}) *contentful.Field {
	contentfulField := &contentful.Field{
		ID:        newField["id"].(string),
		Name:      newField["name"].(string),
		Type:      newField["type"].(string),
		Localized: newField["localized"].(bool),
		Required:  newField["required"].(bool),
		Disabled:  newField["disabled"].(bool),
		Omitted:   newField["omitted"].(bool),
	}

	if linkType, ok := newField["link_type"].(string); ok {
		contentfulField.LinkType = linkType
	}

	if validations, ok := newField["validations"].([]interface{}); ok {
		parsedValidations, _ := contentful.ParseValidations(validations)

		contentfulField.Validations = parsedValidations
	}

	if items := processItems(newField["items"].([]interface{})); items != nil {
		contentfulField.Items = items
	}
	return contentfulField
}

func processItems(fieldItems []interface{}) *contentful.FieldTypeArrayItem {
	var items *contentful.FieldTypeArrayItem

	for i := 0; i < len(fieldItems); i++ {
		item := fieldItems[i].(map[string]interface{})

		var validations []contentful.FieldValidation

		if fieldValidations, ok := item["validations"].([]interface{}); ok {
			validations, _ = contentful.ParseValidations(fieldValidations)
		}

		items = &contentful.FieldTypeArrayItem{
			Type:        item["type"].(string),
			Validations: validations,
			LinkType:    item["link_type"].(string),
		}
	}
	return items
}
