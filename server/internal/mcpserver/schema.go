package mcpserver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

const refPrefix = "#/components/schemas/"

func findOperation(doc *huma.OpenAPI, id string) *huma.Operation {
	for _, item := range doc.Paths {
		for _, op := range []*huma.Operation{item.Get, item.Post, item.Put, item.Patch, item.Delete} {
			if op != nil && op.OperationID == id {
				return op
			}
		}
	}
	return nil
}

// inputSchema builds a tool's input schema: one object holding the
// operation's path and query parameters and the fields of its body. Schemas
// the body refers to are carried along as $defs.
func inputSchema(doc *huma.OpenAPI, op *huma.Operation) (map[string]any, error) {
	properties := map[string]any{}
	var required []string
	defs := map[string]any{}

	for _, param := range op.Parameters {
		if param.In != "path" && param.In != "query" {
			continue
		}
		schema, err := toMap(param.Schema)
		if err != nil {
			return nil, err
		}
		description := param.Description
		if description == "" && param.In == "path" {
			description = pathParamDescription(op.Path, param.Name)
		}
		if description != "" {
			schema["description"] = description
		}
		properties[param.Name] = schema
		if param.Required || param.In == "path" {
			required = append(required, param.Name)
		}
	}

	if op.RequestBody != nil {
		content, ok := op.RequestBody.Content["application/json"]
		if !ok || content.Schema == nil {
			return nil, fmt.Errorf("the body is not JSON")
		}
		body, err := toMap(content.Schema)
		if err != nil {
			return nil, err
		}
		if ref, ok := body["$ref"].(string); ok {
			body, err = componentMap(doc, strings.TrimPrefix(ref, refPrefix))
			if err != nil {
				return nil, err
			}
		}
		bodyProps, _ := body["properties"].(map[string]any)
		for name, schema := range bodyProps {
			if _, taken := properties[name]; taken {
				return nil, fmt.Errorf("the body field %q collides with a parameter", name)
			}
			properties[name] = schema
		}
		if bodyRequired, ok := body["required"].([]any); ok {
			for _, name := range bodyRequired {
				required = append(required, fmt.Sprint(name))
			}
		}
	}

	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}

	// Carry along every schema reachable through a $ref, with the refs
	// pointed at $defs.
	var pending []string
	collectRefs(schema, &pending)
	for len(pending) > 0 {
		name := pending[0]
		pending = pending[1:]
		if _, done := defs[name]; done {
			continue
		}
		component, err := componentMap(doc, name)
		if err != nil {
			return nil, err
		}
		defs[name] = component
		collectRefs(component, &pending)
	}
	if len(defs) > 0 {
		schema["$defs"] = defs
	}
	rewriteRefs(schema)
	return schema, nil
}

func pathParamDescription(path, name string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if segment == "{"+name+"}" && i > 0 {
			resource := strings.TrimSuffix(strings.ReplaceAll(segments[i-1], "-", " "), "s")
			return "The " + resource + " id."
		}
	}
	return ""
}

func componentMap(doc *huma.OpenAPI, name string) (map[string]any, error) {
	schema, ok := doc.Components.Schemas.Map()[name]
	if !ok {
		return nil, fmt.Errorf("unknown schema %q", name)
	}
	return toMap(schema)
}

// toMap round trips a schema through JSON, giving a plain copy that can be
// edited without touching the live document.
func toMap(schema *huma.Schema) (map[string]any, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	// The schema of the schema is noise to a model.
	delete(out, "$schema")
	return out, nil
}

func collectRefs(value any, into *[]string) {
	switch v := value.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok && strings.HasPrefix(ref, refPrefix) {
			*into = append(*into, strings.TrimPrefix(ref, refPrefix))
		}
		for key, child := range v {
			if key != "$defs" {
				collectRefs(child, into)
			}
		}
	case []any:
		for _, child := range v {
			collectRefs(child, into)
		}
	}
}

func rewriteRefs(value any) {
	switch v := value.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok && strings.HasPrefix(ref, refPrefix) {
			v["$ref"] = "#/$defs/" + strings.TrimPrefix(ref, refPrefix)
		}
		for _, child := range v {
			rewriteRefs(child)
		}
	case []any:
		for _, child := range v {
			rewriteRefs(child)
		}
	}
}
