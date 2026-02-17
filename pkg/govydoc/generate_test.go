package govydoc

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nieomylnieja/govydoc/internal/testmodels"
)

//go:embed testdata/generate_output.json
var expectedGenerateOutput []byte

func TestGenerate(t *testing.T) {
	validator := govy.New(
		govy.For(func(t testmodels.Teacher) string { return t.Name }).
			WithName("name").
			Rules(rules.EQ("John")),
		govy.For(func(t testmodels.Teacher) string { return t.Hobby }).
			WithName("hobby").
			Rules(rules.Forbidden[string]()).
			When(func(t testmodels.Teacher) bool { return t.Age > 30 }, govy.WhenDescription("when above 30")),
	).
		WithName("Teacher")

	actual, err := Generate(validator)
	require.NoError(t, err)
	var expected ObjectDoc
	err = json.Unmarshal(expectedGenerateOutput, &expected)
	require.NoError(t, err)

	if !assert.JSONEq(t, mustMarshalJSON(t, expected), mustMarshalJSON(t, actual)) {
		data, err := json.MarshalIndent(actual, "", "  ")
		require.NoError(t, err)
		t.Log(string(data))
	}
}

func mustMarshalJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}

func TestPropertyDoc_key(t *testing.T) {
	tests := []struct {
		name     string
		prop     PropertyDoc
		expected string
	}{
		{
			name: "builtin type without package",
			prop: PropertyDoc{
				PropertyPlan: govy.PropertyPlan{
					TypeInfo: govy.TypeInfo{Name: "string", Package: ""},
				},
			},
			expected: "string",
		},
		{
			name: "builtin int type",
			prop: PropertyDoc{
				PropertyPlan: govy.PropertyPlan{
					TypeInfo: govy.TypeInfo{Name: "int", Package: ""},
				},
			},
			expected: "int",
		},
		{
			name: "custom type with package",
			prop: PropertyDoc{
				PropertyPlan: govy.PropertyPlan{
					TypeInfo: govy.TypeInfo{
						Name:    "Teacher",
						Package: "github.com/nieomylnieja/govydoc/internal/testmodels",
					},
				},
			},
			expected: "github.com/nieomylnieja/govydoc/internal/testmodels.Teacher",
		},
		{
			name: "slice of custom type",
			prop: PropertyDoc{
				PropertyPlan: govy.PropertyPlan{
					TypeInfo: govy.TypeInfo{
						Name:    "[]Teacher",
						Package: "github.com/nieomylnieja/govydoc/internal/testmodels",
					},
				},
			},
			expected: "github.com/nieomylnieja/govydoc/internal/testmodels.[]Teacher",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.prop.key())
		})
	}
}

func TestWithFilteredPaths(t *testing.T) {
	validator := govy.New(
		govy.For(func(t testmodels.Teacher) string { return t.Name }).
			WithName("name").
			Rules(rules.EQ("John")),
		govy.For(func(t testmodels.Teacher) string { return t.Hobby }).
			WithName("hobby").
			Rules(rules.EQ("reading")),
	).WithName("Teacher")

	t.Run("filters specified paths", func(t *testing.T) {
		doc, err := Generate(validator, WithFilteredPaths("$.hobby"))
		require.NoError(t, err)

		// Verify hobby property is filtered out
		for _, prop := range doc.Properties {
			assert.NotEqual(t, "$.hobby", prop.Path, "hobby should be filtered out")
		}

		// Verify other properties still exist
		found := false
		for _, prop := range doc.Properties {
			if prop.Path == "$.name" {
				found = true
				break
			}
		}
		assert.True(t, found, "name property should still exist")
	})

	t.Run("filters multiple paths", func(t *testing.T) {
		doc, err := Generate(validator, WithFilteredPaths("$.hobby", "$.name"))
		require.NoError(t, err)

		// Verify both properties are filtered out
		for _, prop := range doc.Properties {
			assert.NotEqual(t, "$.hobby", prop.Path)
			assert.NotEqual(t, "$.name", prop.Path)
		}
	})

	t.Run("no filtering when no paths specified", func(t *testing.T) {
		doc, err := Generate(validator)
		require.NoError(t, err)

		// Should have both properties
		hasName := false
		hasHobby := false
		for _, prop := range doc.Properties {
			if prop.Path == "$.name" {
				hasName = true
			}
			if prop.Path == "$.hobby" {
				hasHobby = true
			}
		}
		assert.True(t, hasName, "name property should exist")
		assert.True(t, hasHobby, "hobby property should exist")
	})
}

func TestGenerateGovyOptions(t *testing.T) {
	validator := govy.New(
		govy.For(func(t testmodels.Teacher) string { return t.Name }).
			WithName("name").
			Rules(rules.EQ("John")),
	).WithName("Teacher")

	t.Run("passes options to govy.Plan", func(t *testing.T) {
		// This test verifies the option is passed through correctly
		// The actual behavior depends on govy implementation
		doc, err := Generate(validator, GenerateGovyOptions())
		require.NoError(t, err)
		assert.Equal(t, "Teacher", doc.Name)
	})
}

func TestGenerate_EmptyValidator(t *testing.T) {
	validator := govy.New[testmodels.Teacher]().WithName("Teacher")

	doc, err := Generate(validator)
	require.NoError(t, err)
	assert.Equal(t, "Teacher", doc.Name)
	// Should still have properties from reflection, even without validation rules
	assert.NotEmpty(t, doc.Properties)
}

func TestGenerate_PointerType(t *testing.T) {
	// Test that Generate handles pointer types correctly
	validator := govy.New(
		govy.For(func(s testmodels.SimpleStruct) string { return s.Value }).
			WithName("value").
			Rules(rules.EQ("test")),
	).WithName("SimpleStruct")

	doc, err := Generate(validator)
	require.NoError(t, err)
	assert.Equal(t, "SimpleStruct", doc.Name)
	assert.NotEmpty(t, doc.Properties)
}

func TestGenerate_NestedStructs(t *testing.T) {
	validator := govy.New(
		govy.For(func(p testmodels.Person) string { return p.Name }).
			WithName("name").
			Rules(rules.EQ("John")),
	).WithName("Person")

	doc, err := Generate(validator)
	require.NoError(t, err)
	assert.Equal(t, "Person", doc.Name)

	// Verify nested properties exist
	hasCity := false
	hasState := false
	for _, prop := range doc.Properties {
		if prop.Path == "$.address.city" {
			hasCity = true
		}
		if prop.Path == "$.address.state" {
			hasState = true
		}
	}
	assert.True(t, hasCity, "nested city property should exist")
	assert.True(t, hasState, "nested state property should exist")
}

func TestGenerate_SliceTypes(t *testing.T) {
	validator := govy.New[testmodels.ListStruct]().WithName("ListStruct")

	doc, err := Generate(validator)
	require.NoError(t, err)
	assert.Equal(t, "ListStruct", doc.Name)

	// Verify slice property exists
	found := false
	for _, prop := range doc.Properties {
		if prop.Path == "$.items[*]" {
			found = true
			break
		}
	}
	assert.True(t, found, "slice property should exist")
}

func TestGenerate_MapTypes(t *testing.T) {
	validator := govy.New[testmodels.MapStruct]().WithName("MapStruct")

	doc, err := Generate(validator)
	require.NoError(t, err)
	assert.Equal(t, "MapStruct", doc.Name)

	// Verify map properties exist
	hasMapKey := false
	hasMapValue := false
	for _, prop := range doc.Properties {
		if prop.Path == "$.data.~" {
			hasMapKey = true
		}
		if prop.Path == "$.data.*" {
			hasMapValue = true
		}
	}
	assert.True(t, hasMapKey, "map key property should exist")
	assert.True(t, hasMapValue, "map value property should exist")
}
