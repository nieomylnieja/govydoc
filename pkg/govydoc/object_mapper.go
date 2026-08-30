package govydoc

import (
	"reflect"
	"strings"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/jsonpath"

	"github.com/nieomylnieja/govydoc/internal/typeinfo"
)

type objectMapper struct {
	properties      []PropertyDoc
	opaqueTypeKinds map[reflect.Type]string
	opaquePathKinds map[string]string
}

func newObjectMapper(opaqueTypeKinds map[reflect.Type]string) *objectMapper {
	return &objectMapper{
		opaqueTypeKinds: opaqueTypeKinds,
		opaquePathKinds: make(map[string]string),
	}
}

func (o *objectMapper) mapType(typ reflect.Type, path jsonpath.Path) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	doc := PropertyDoc{}
	doc.Path = path
	doc = setTypeInfo(doc, typ)
	if kind, ok := o.opaqueTypeKinds[typ]; ok {
		o.properties = append(o.properties, doc)
		o.opaquePathKinds[path.String()] = kind
		return
	}
	o.properties = append(o.properties, doc)

	switch typ.Kind() {
	case reflect.Struct:
		for _, field := range reflect.VisibleFields(typ) {
			if !field.IsExported() {
				continue
			}
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "" || name == "-" {
				continue
			}
			o.mapType(field.Type, path.Name(name))
		}
	case reflect.Slice:
		o.mapType(typ.Elem(), path.IndexWildcard())
	case reflect.Map:
		o.mapType(typ.Key(), path.KeyWildcard())
		o.mapType(typ.Elem(), path.ValueWildcard())
	default:
	}
}

func setTypeInfo(doc PropertyDoc, typ reflect.Type) PropertyDoc {
	doc.TypeInfo = govy.TypeInfo(typeinfo.Get(typ))
	return doc
}
