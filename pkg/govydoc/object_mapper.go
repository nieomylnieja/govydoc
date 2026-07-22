package govydoc

import (
	"reflect"
	"strings"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/jsonpath"

	"github.com/nieomylnieja/govydoc/internal/typeinfo"
)

func newObjectMapper() *objectMapper {
	return &objectMapper{}
}

type objectMapper struct {
	Properties []PropertyDoc
}

func (o *objectMapper) Map(typ reflect.Type, path jsonpath.Path) {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	doc := PropertyDoc{}
	doc.Path = path
	doc = setTypeInfo(doc, typ)
	o.Properties = append(o.Properties, doc)

	switch typ.Kind() {
	case reflect.Struct:
		for _, field := range reflect.VisibleFields(typ) {
			tags := strings.Split(field.Tag.Get("json"), ",")
			if len(tags) == 0 {
				continue
			}
			name := tags[0]
			if name == "" || name == "-" {
				continue
			}
			o.Map(field.Type, path.Name(name))
		}
	case reflect.Slice:
		o.Map(typ.Elem(), path.IndexWildcard())
	case reflect.Map:
		o.Map(typ.Key(), path.KeyWildcard())
		o.Map(typ.Elem(), path.ValueWildcard())
	default:
	}
}

func setTypeInfo(doc PropertyDoc, typ reflect.Type) PropertyDoc {
	doc.TypeInfo = govy.TypeInfo(typeinfo.Get(typ))
	return doc
}
