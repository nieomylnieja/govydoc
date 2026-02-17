package govydoc

import (
	"regexp"
	"slices"
	"strings"
)

// filterProperties is a list of property paths that should be filtered out from the documentation.
var filterProperties = []string{
	"$.organization",
}

func postProcessProperties(doc ObjectDoc, formatters ...propertyPostProcessor) ObjectDoc {
	properties := make([]PropertyDoc, 0, len(doc.Properties))
	for _, property := range doc.Properties {
		if slices.Contains(filterProperties, property.Path) {
			continue
		}
		for _, formatter := range formatters {
			property = formatter(property)
		}
		properties = append(properties, property)
	}
	doc.Properties = properties
	return doc
}

// propertyPostProcessor is a function type that post-processes PropertyDoc.
// It can be used to apply additional formatting to the property documentation or add more details to the doc.
type propertyPostProcessor func(doc PropertyDoc) PropertyDoc

var (
	enumDeclarationRegex = regexp.MustCompile(`(?s)ENUM(.*)`)
	deprecatedRegex      = regexp.MustCompile(`(?m)^Deprecated:\s*(.*)$`)
)

// removeEnumDeclaration removes ENUM (used with go-enum generator) declarations from the code docs.
func removeEnumDeclaration(doc PropertyDoc) PropertyDoc {
	doc.TypeDoc = enumDeclarationRegex.ReplaceAllString(doc.TypeDoc, "")
	return doc
}

// removeTrailingWhitespace removes trailing whitespace from the docs.
func removeTrailingWhitespace(doc PropertyDoc) PropertyDoc {
	doc.TypeDoc = strings.TrimSpace(doc.TypeDoc)
	doc.FieldDoc = strings.TrimSpace(doc.FieldDoc)
	return doc
}

// extractDeprecatedInformation extracts deprecated information from the docs
// and sets PropertyDoc.DeprecatedDoc accordingly.
func extractDeprecatedInformation(doc PropertyDoc) PropertyDoc {
	switch {
	case deprecatedRegex.MatchString(doc.TypeDoc):
		matches := deprecatedRegex.FindStringSubmatch(doc.TypeDoc)
		if len(matches) > 1 {
			doc.DeprecatedDoc = strings.TrimSpace(matches[1])
		}
		doc.TypeDoc = strings.TrimSpace(deprecatedRegex.ReplaceAllString(doc.TypeDoc, ""))
	case deprecatedRegex.MatchString(doc.FieldDoc):
		matches := deprecatedRegex.FindStringSubmatch(doc.FieldDoc)
		if len(matches) > 1 {
			doc.DeprecatedDoc = strings.TrimSpace(matches[1])
		}
		doc.FieldDoc = strings.TrimSpace(deprecatedRegex.ReplaceAllString(doc.FieldDoc, ""))
	}
	return doc
}
