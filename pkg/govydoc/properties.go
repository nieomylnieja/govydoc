package govydoc

import (
	"regexp"
	"slices"
	"strings"

	"github.com/nobl9/govy/pkg/jsonpath"
)

var (
	enumDeclarationRegex = regexp.MustCompile(`(?s)ENUM(.*)`)
	deprecatedRegex      = regexp.MustCompile(`(?m)^Deprecated:\s*(.*)$`)
)

type propertyPostProcessor func(doc PropertyDoc) PropertyDoc

func postProcessProperties(doc ObjectDoc, filterPaths []jsonpath.Path, formatters ...propertyPostProcessor) ObjectDoc {
	properties := make([]PropertyDoc, 0, len(doc.Properties))
	for _, property := range doc.Properties {
		if containsPath(filterPaths, property.Path) {
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

func containsPath(paths []jsonpath.Path, path jsonpath.Path) bool {
	return slices.ContainsFunc(paths, func(candidate jsonpath.Path) bool {
		return candidate.Equal(path)
	})
}

func removeEnumDeclaration(doc PropertyDoc) PropertyDoc {
	doc.TypeDoc = enumDeclarationRegex.ReplaceAllString(doc.TypeDoc, "")
	return doc
}

func removeTrailingWhitespace(doc PropertyDoc) PropertyDoc {
	doc.TypeDoc = strings.TrimSpace(doc.TypeDoc)
	doc.FieldDoc = strings.TrimSpace(doc.FieldDoc)
	return doc
}

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
