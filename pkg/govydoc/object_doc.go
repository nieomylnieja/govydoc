package govydoc

import (
	"reflect"
	"strings"

	"github.com/nobl9/govy/pkg/jsonpath"
)

func generateObjectDoc(
	goType reflect.Type,
	opaqueTypeKinds map[reflect.Type]string,
) (doc ObjectDoc, opaquePathKinds map[string]string) {
	for goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}
	mapper := newObjectMapper(opaqueTypeKinds)
	mapper.mapType(goType, jsonpath.Parse("$"))

	doc = ObjectDoc{
		Properties: mapper.properties,
	}
	for i, property := range doc.Properties {
		childrenPaths := findPropertyChildrenPaths(property.Path, doc.Properties)
		property.ChildrenPaths = childrenPaths
		doc.Properties[i] = property
	}
	return doc, mapper.opaquePathKinds
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
