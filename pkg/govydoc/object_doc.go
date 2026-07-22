package govydoc

import (
	"reflect"
	"strings"

	"github.com/nobl9/govy/pkg/jsonpath"
)

func generateObjectDoc(goType reflect.Type) ObjectDoc {
	for goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}
	mapper := newObjectMapper()
	mapper.mapType(goType, jsonpath.Parse("$"))

	objectDoc := ObjectDoc{
		Properties: mapper.properties,
	}
	for i, property := range objectDoc.Properties {
		childrenPaths := findPropertyChildrenPaths(property.Path, objectDoc.Properties)
		property.ChildrenPaths = childrenPaths
		objectDoc.Properties[i] = property
	}
	return objectDoc
}

func findPropertyChildrenPaths(parent jsonpath.Path, properties []PropertyDoc) []string {
	childrenPaths := make([]string, 0, len(properties))
	parentString := parent.String()
	for _, property := range properties {
		childRelativePath, found := strings.CutPrefix(property.Path.String(), parentString+".")
		if !found {
			continue
		}
		if strings.Contains(childRelativePath, ".") {
			continue
		}
		childrenPaths = append(childrenPaths, parentString+"."+childRelativePath)
	}
	return childrenPaths
}
