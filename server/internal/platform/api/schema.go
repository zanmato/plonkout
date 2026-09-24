package api

import (
	"slices"

	"github.com/danielgtaylor/huma/v2"
)

// FixNullableEnums adds null to the enum of every nullable enum, e.g. a
// *string intensity that is "heavy", "light" or null. Huma marks the type
// nullable but leaves the enum as declared, which JSON Schema reads as "never
// null" and client generators turn into a type without null.
func FixNullableEnums(doc *huma.OpenAPI) {
	if doc.Components == nil || doc.Components.Schemas == nil {
		return
	}
	for _, schema := range doc.Components.Schemas.Map() {
		fixSchema(schema)
	}
}

func fixSchema(s *huma.Schema) {
	if s == nil {
		return
	}
	if s.Nullable && len(s.Enum) > 0 && !slices.Contains(s.Enum, any(nil)) {
		s.Enum = append(s.Enum, nil)
	}
	for _, property := range s.Properties {
		fixSchema(property)
	}
	fixSchema(s.Items)
}
