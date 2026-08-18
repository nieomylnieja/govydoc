package typeinfo

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		typ      reflect.Type
		expected TypeInfo
	}{
		"nil": {
			expected: TypeInfo{},
		},
		"int": {
			typ:      reflect.TypeFor[int](),
			expected: TypeInfo{Name: "int", Kind: "int"},
		},
		"pointer to int": {
			typ:      reflect.TypeFor[*int](),
			expected: TypeInfo{Name: "int", Kind: "int"},
		},
		"nested pointer to int": {
			typ:      reflect.TypeFor[***int](),
			expected: TypeInfo{Name: "int", Kind: "int"},
		},
		"slice of int": {
			typ:      reflect.TypeFor[[]int](),
			expected: TypeInfo{Name: "[]int", Kind: "[]int"},
		},
		"slice of custom string": {
			typ:      reflect.TypeFor[[]customString](),
			expected: TypeInfo{Name: "[]customString", Package: packageName, Kind: "[]string"},
		},
		"map of string to int": {
			typ:      reflect.TypeFor[map[string]int](),
			expected: TypeInfo{Name: "map[string]int", Kind: "map[string]int"},
		},
		"custom string": {
			typ:      reflect.TypeFor[customString](),
			expected: TypeInfo{Name: "customString", Package: packageName, Kind: "string"},
		},
		"custom struct": {
			typ:      reflect.TypeFor[customStruct](),
			expected: TypeInfo{Name: "customStruct", Package: packageName, Kind: "struct"},
		},
		"pointer to custom struct": {
			typ:      reflect.TypeFor[*customStruct](),
			expected: TypeInfo{Name: "customStruct", Package: packageName, Kind: "struct"},
		},
		"custom map": {
			typ:      reflect.TypeFor[customMap](),
			expected: TypeInfo{Name: "customMap", Package: packageName, Kind: "map[string]int"},
		},
		"custom nested map": {
			typ: reflect.TypeFor[customNestedMap](),
			expected: TypeInfo{
				Name:    "customNestedMap",
				Package: packageName,
				Kind:    "map[string][]map[string]int",
			},
		},
		"custom slice": {
			typ:      reflect.TypeFor[customSlice](),
			expected: TypeInfo{Name: "customSlice", Package: packageName, Kind: "[]map[string]int"},
		},
		"custom string slice": {
			typ:      reflect.TypeFor[customStringSlice](),
			expected: TypeInfo{Name: "customStringSlice", Package: packageName, Kind: "[]string"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, Get(test.typ))
		})
	}
}

const packageName = "github.com/nieomylnieja/govydoc/internal/typeinfo"

type customString string

type customStruct struct{}

type customMap map[string]int

type customSlice []customMap

type customStringSlice []string

type customNestedMap map[customString]customSlice
