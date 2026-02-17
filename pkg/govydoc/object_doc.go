package govydoc

import (
	"reflect"
	"strings"
)

func generateObjectDoc(goType reflect.Type) ObjectDoc {
	if goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}
	// Generate object properties based on reflection.
	mapper := newObjectMapper()
	mapper.Map(goType, "$")

	objectDoc := ObjectDoc{
		Properties: mapper.Properties,
	}
	// Add children paths to properties.
	// The object mapper does not provide this information, but rather returns a flat list of properties.
	for i, property := range objectDoc.Properties {
		childrenPaths := findPropertyChildrenPaths(property.Path, objectDoc.Properties)
		property.ChildrenPaths = childrenPaths
		objectDoc.Properties[i] = property
	}
	return objectDoc
}

func findPropertyChildrenPaths(parent string, properties []PropertyDoc) []string {
	childrenPaths := make([]string, 0, len(properties))
	for _, property := range properties {
		childRelativePath, found := strings.CutPrefix(property.Path, parent+".")
		if !found {
			continue
		}
		// Not an immediate child.
		if strings.Contains(childRelativePath, ".") {
			continue
		}
		childrenPaths = append(childrenPaths, parent+"."+childRelativePath)
	}
	return childrenPaths
}
