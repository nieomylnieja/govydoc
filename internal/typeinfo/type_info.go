package typeinfo

import "reflect"

// TypeInfo stores the Go type information.
type TypeInfo struct {
	Name    string
	Kind    string
	Package string
}

// Get returns information about typ with pointer layers removed.
// Built-in types have an empty package, while slices of named types keep the slice notation in their name.
func Get(typ reflect.Type) TypeInfo {
	if typ == nil {
		return TypeInfo{}
	}
	for typ.Kind() == reflect.Pointer {
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
		return "map[" + getKindString(typ.Key()) + "]" + getKindString(typ.Elem())
	case reflect.Slice:
		return "[]" + getKindString(typ.Elem())
	default:
		return typ.Kind().String()
	}
}
