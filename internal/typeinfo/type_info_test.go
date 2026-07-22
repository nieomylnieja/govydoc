package typeinfo

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const packageName = "github.com/nieomylnieja/govydoc/internal/typeinfo"

type customString string

type customStruct struct{}

type customMap map[string]int

type customSlice []customMap

type customStringSlice []string

type customNestedMap map[customString]customSlice

type testCase struct {
	name     string
	value    any
	expected TypeInfo
}

func TestGet(t *testing.T) {
	tests := []testCase{
		{
			name:     "int",
			value:    0,
			expected: TypeInfo{Name: "int", Package: "", Kind: "int"},
		},
		{
			name:     "pointer to int",
			value:    new(int),
			expected: TypeInfo{Name: "int", Package: "", Kind: "int"},
		},
		{
			name:     "slice of int",
			value:    []int{},
			expected: TypeInfo{Name: "[]int", Package: "", Kind: "[]int"},
		},
		{
			name:     "slice of customString",
			value:    []customString{},
			expected: TypeInfo{Name: "[]customString", Package: packageName, Kind: "[]string"},
		},
		{
			name:     "map of string to int",
			value:    map[string]int{},
			expected: TypeInfo{Name: "map[string]int", Package: "", Kind: "map[string]int"},
		},
		{
			name:     "custom string",
			value:    customString(""),
			expected: TypeInfo{Name: "customString", Package: packageName, Kind: "string"},
		},
		{
			name:     "custom struct",
			value:    customStruct{},
			expected: TypeInfo{Name: "customStruct", Package: packageName, Kind: "struct"},
		},
		{
			name:     "pointer to custom struct",
			value:    &customStruct{},
			expected: TypeInfo{Name: "customStruct", Package: packageName, Kind: "struct"},
		},
		{
			name:     "custom map",
			value:    customMap{},
			expected: TypeInfo{Name: "customMap", Package: packageName, Kind: "map[string]int"},
		},
		{
			name:     "custom nested map",
			value:    customNestedMap{},
			expected: TypeInfo{Name: "customNestedMap", Package: packageName, Kind: "map[string][]map[string]int"},
		},
		{
			name:     "custom slice",
			value:    customSlice{},
			expected: TypeInfo{Name: "customSlice", Package: packageName, Kind: "[]map[string]int"},
		},
		{
			name:     "custom string slice",
			value:    customStringSlice{},
			expected: TypeInfo{Name: "customStringSlice", Package: packageName, Kind: "[]string"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := reflect.TypeOf(tc.value)
			actual := Get(rt)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
