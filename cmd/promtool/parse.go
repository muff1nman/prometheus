package main

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const maxRecursion = 500

func ParseSchemaFromBytes(input []byte, visitor schemaVisitorFn) (reparsedSchemaBytes []byte, err error) {
	var schema apiextensionsv1.JSONSchemaProps
	err = json.Unmarshal(input, &schema)
	if err != nil {
		return nil, err
	}

	reparsedSchema, err := ParseSchema(schema, visitor)
	if err != nil {
		return nil, err
	}

	return json.Marshal(reparsedSchema)
}

func ParseSchema(schema apiextensionsv1.JSONSchemaProps, visitor schemaVisitorFn) (apiextensionsv1.JSONSchemaProps, error) {
	schema.Schema = "http://json-schema.org/draft-04/schema#"

	if visitor != nil {
		visitSchema(&visitCtx{}, &schema, visitor)
	}
	return schema, nil
}

type schemaVisitorFn func(schema *apiextensionsv1.JSONSchemaProps)

type visitCtx struct {
	recursionLevel int
}

func visitSchema(ctx *visitCtx, schema *apiextensionsv1.JSONSchemaProps, visit schemaVisitorFn) {
	ctx.recursionLevel++
	if ctx.recursionLevel > maxRecursion {
		return
	}
	defer func() { ctx.recursionLevel-- }()

	visitSlice := func(props []apiextensionsv1.JSONSchemaProps) {
		for i, s := range props {
			s := s
			visitSchema(ctx, &s, visit)
			props[i] = s
		}
	}
	visitMap := func(props map[string]apiextensionsv1.JSONSchemaProps) {
		for k, s := range props {
			s := s
			visitSchema(ctx, &s, visit)
			props[k] = s
		}
	}

	if schema == nil {
		return
	}

	// First visit the current schema props.
	visit(schema)

	visitSlice(schema.AllOf)
	visitSlice(schema.OneOf)
	visitSlice(schema.AnyOf)
	visitSchema(ctx, schema.Not, visit)
	visitMap(schema.Properties)
	visitMap(schema.PatternProperties)
	if schema.AdditionalProperties != nil {
		visitSchema(ctx, schema.AdditionalProperties.Schema, visit)
	}
	if schema.AdditionalItems != nil {
		visitSchema(ctx, schema.AdditionalItems.Schema, visit)
	}

	// Then recurse into child schema props.
	if schema.Items != nil {
		visitSchema(ctx, schema.Items.Schema, visit)
		visitSlice(schema.Items.JSONSchemas)
	}
	if schema.Definitions != nil {
		visitMap(schema.Definitions)
	}
}
