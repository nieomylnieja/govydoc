package govydoc

import (
	_ "embed"
	"encoding/json"
	"sync"
	"testing"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nieomylnieja/govydoc/internal/testmodels"
)

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
	require.NoError(t, json.Unmarshal(expectedGenerateOutput, &expected))

	if !assert.JSONEq(t, mustMarshalJSON(t, expected), mustMarshalJSON(t, actual)) {
		data, err := json.MarshalIndent(actual, "", "  ")
		require.NoError(t, err)
		t.Log(string(data))
	}
}

func TestPropertyDoc_key(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		property PropertyDoc
		expected string
	}{
		"built-in string": {
			property: PropertyDoc{PropertyPlan: govy.PropertyPlan{
				TypeInfo: govy.TypeInfo{Name: "string"},
			}},
			expected: "string",
		},
		"custom type": {
			property: PropertyDoc{PropertyPlan: govy.PropertyPlan{
				TypeInfo: govy.TypeInfo{
					Name:    "Teacher",
					Package: "github.com/nieomylnieja/govydoc/internal/testmodels",
				},
			}},
			expected: "github.com/nieomylnieja/govydoc/internal/testmodels.Teacher",
		},
		"slice of custom type": {
			property: PropertyDoc{PropertyPlan: govy.PropertyPlan{
				TypeInfo: govy.TypeInfo{
					Name:    "[]Teacher",
					Package: "github.com/nieomylnieja/govydoc/internal/testmodels",
				},
			}},
			expected: "github.com/nieomylnieja/govydoc/internal/testmodels.[]Teacher",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, test.property.key())
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
	).
		WithName("Teacher")

	t.Run("one path", func(t *testing.T) {
		doc, err := Generate(validator, WithFilteredPaths("$.hobby"))
		require.NoError(t, err)

		paths := propertyPaths(doc)
		assert.NotContains(t, paths, "$.hobby")
		assert.Contains(t, paths, "$.name")
	})

	t.Run("multiple paths", func(t *testing.T) {
		doc, err := Generate(validator, WithFilteredPaths("$.hobby", "$.name"))
		require.NoError(t, err)

		paths := propertyPaths(doc)
		assert.NotContains(t, paths, "$.hobby")
		assert.NotContains(t, paths, "$.name")
	})

	t.Run("no paths", func(t *testing.T) {
		doc, err := Generate(validator)
		require.NoError(t, err)

		paths := propertyPaths(doc)
		assert.Contains(t, paths, "$.name")
		assert.Contains(t, paths, "$.hobby")
	})
}

func TestGenerateGovyOptions(t *testing.T) {
	validator := govy.New(
		govy.For(func(t testmodels.Teacher) string { return t.Name }).
			WithName("name").
			Rules(rules.EQ("John")),
	).
		WithName("Teacher")

	doc, err := Generate(validator, GenerateGovyOptions())

	require.NoError(t, err)
	assert.Equal(t, "Teacher", doc.Name)
}

func TestGenerate_EmptyValidator(t *testing.T) {
	validator := govy.New[testmodels.Teacher]().WithName("Teacher")

	doc, err := Generate(validator)

	require.NoError(t, err)
	assert.Equal(t, "Teacher", doc.Name)
	assert.NotEmpty(t, doc.Properties)
}

func TestGenerate_PointerType(t *testing.T) {
	validator := govy.New(
		govy.For(func(s *testmodels.SimpleStruct) string { return s.Value }).
			WithName("value").
			Rules(rules.EQ("test")),
	).
		WithName("SimpleStruct")

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
	).
		WithName("Person")

	doc, err := Generate(validator)

	require.NoError(t, err)
	assert.Equal(t, "Person", doc.Name)
	paths := propertyPaths(doc)
	assert.Contains(t, paths, "$.address.city")
	assert.Contains(t, paths, "$.address.state")
}

func TestGenerate_SliceTypes(t *testing.T) {
	validator := govy.New[testmodels.ListStruct]().WithName("ListStruct")

	doc, err := Generate(validator)

	require.NoError(t, err)
	assert.Equal(t, "ListStruct", doc.Name)
	assert.Contains(t, propertyPaths(doc), "$.items[*]")
}

func TestGenerate_MapTypes(t *testing.T) {
	validator := govy.New[testmodels.MapStruct]().WithName("MapStruct")

	doc, err := Generate(validator)

	require.NoError(t, err)
	assert.Equal(t, "MapStruct", doc.Name)
	paths := propertyPaths(doc)
	assert.Contains(t, paths, "$.data.*~")
	assert.Contains(t, paths, "$.data.*")
}

func TestWithGenerator_ConcurrentReuse(t *testing.T) {
	teacherValidator := govy.New[testmodels.Teacher]().WithName("Teacher")
	personValidator := govy.New[testmodels.Person]().WithName("Person")
	expectedTeacher, err := Generate(teacherValidator)
	require.NoError(t, err)
	expectedPerson, err := Generate(personValidator)
	require.NoError(t, err)

	generator, err := NewGenerator()
	require.NoError(t, err)
	option := WithGenerator(generator)

	generate := []func() (ObjectDoc, error){
		func() (ObjectDoc, error) {
			return Generate(teacherValidator, option)
		},
		func() (ObjectDoc, error) {
			return Generate(personValidator, option)
		},
	}

	docs := make([]ObjectDoc, len(generate))
	errs := make([]error, len(generate))
	var waitGroup sync.WaitGroup
	for i, generateDoc := range generate {
		waitGroup.Go(func() {
			docs[i], errs[i] = generateDoc()
		})
	}
	waitGroup.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.JSONEq(t, mustMarshalJSON(t, expectedTeacher), mustMarshalJSON(t, docs[0]))
	assert.JSONEq(t, mustMarshalJSON(t, expectedPerson), mustMarshalJSON(t, docs[1]))
}

func TestWithGenerator_Nil(t *testing.T) {
	validator := govy.New[testmodels.Teacher]().WithName("Teacher")

	_, err := Generate(validator, WithGenerator(nil))

	require.EqualError(t, err, "generator cannot be nil")
}

func TestGenerate_PromotedFieldDocumentation(t *testing.T) {
	validator := govy.New[testmodels.StructWithPromotedFields]().WithName("StructWithPromotedFields")

	doc, err := Generate(validator)

	require.NoError(t, err)
	for _, property := range doc.Properties {
		if property.Path.String() == "$.promotedValue" {
			assert.Equal(t, "PromotedValue documents a promoted field.", property.FieldDoc)
			return
		}
	}
	t.Fatal("promoted field property not found")
}

func TestGenerate_DirectFieldDocumentationShadowsPromotedField(t *testing.T) {
	validator := govy.New[testmodels.StructWithShadowedPromotedField]().WithName("StructWithShadowedPromotedField")

	doc, err := Generate(validator)

	require.NoError(t, err)
	for _, property := range doc.Properties {
		if property.Path.String() == "$.promotedValue" {
			assert.Equal(t, "PromotedValue documents the direct field.", property.FieldDoc)
			return
		}
	}
	t.Fatal("direct field property not found")
}

//go:embed testdata/generate_output.json
var expectedGenerateOutput []byte

func mustMarshalJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return string(data)
}

func propertyPaths(doc ObjectDoc) []string {
	paths := make([]string, 0, len(doc.Properties))
	for _, property := range doc.Properties {
		paths = append(paths, property.Path.String())
	}
	return paths
}
