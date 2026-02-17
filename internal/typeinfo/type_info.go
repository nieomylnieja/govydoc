package typeinfo

import (
	"fmt"
	"reflect"
)

// TypeInfo stores the Go type information.
type TypeInfo struct {
	Name    string
	Kind    string
	Package string
}

// Get returns the information for the [reflect.Type].
// It returns TypeInfo containing the type name (without package prefix) and package path separately.
// Strips pointer indicators from type names.
// Package field is empty for built-in types.
//
// Special handling for slices of custom types preserves both the slice notation and the package information.
// Instead of having:
//
//	TypeInfo{Name: "[]mypkg.Bar"}
//
// It will produce:
//
//	TypeInfo{Name: "[]Bar", Package: ".../mypkg"}.
func Get(typ reflect.Type) TypeInfo {
	if typ == nil {
		return TypeInfo{}
	}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	result := TypeInfo{
		Kind: getKindString(typ),
	}

	if typ.PkgPath() == "" && typ.Kind() == reflect.Slice {
		result.Name = "[]"
		typ = typ.Elem()
	}
	switch {
	case typ.PkgPath() == "":
		result.Name += typ.String()
	default:
		result.Name += typ.Name()
		result.Package = typ.PkgPath()
	}
	return result
}

func getKindString(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", getKindString(typ.Key()), getKindString(typ.Elem()))
	case reflect.Slice:
		return fmt.Sprintf("[]%s", getKindString(typ.Elem()))
	default:
		return typ.Kind().String()
	}
}
