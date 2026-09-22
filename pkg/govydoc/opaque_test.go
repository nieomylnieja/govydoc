package govydoc

import (
	"encoding/json"
	"testing"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/jsonpath"
	"github.com/nobl9/govy/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nieomylnieja/govydoc/internal/testmodels"
)

func TestWithOpaqueType_ComponentPlans(t *testing.T) {
	validator := opaqueScalarsValidator()
	plan, err := govy.Plan(validator, govy.PlanStrictMode())
	require.NoError(t, err)

	doc, err := Generate(validator,
		WithOpaqueType[testmodels.OpaqueScalar]("string"),
		GenerateGovyOptions(govy.PlanStrictMode()),
	)
	require.NoError(t, err)

	for _, path := range []string{
		"$.scalar", "$.scalars", "$.pointer", "$.nested.scalar",
		"$.items[*]", "$.mapping.*~", "$.mapping.*", "$['scalar.value']",
	} {
		t.Run(path, func(t *testing.T) {
			assertOpaqueScalarPlans(t, doc, plan, path)
		})
	}

	scalar := requireProperty(t, doc, "$.scalar")
	assert.Equal(t, "OpaqueScalar encodes a numeric value and unit as one string.", scalar.TypeDoc)
	assert.Equal(t, "Scalar is optional even when its internal components are required.", scalar.FieldDoc)
	assert.Empty(t, scalar.Values)
	assert.False(t, scalar.IsHidden)
	assert.Equal(t, []string{"5s"}, scalar.Examples)
	assert.NotContains(t, propertyPaths(doc), "$.secret")
	assert.Empty(t, requireProperty(t, doc, "$").ComponentPlans)
	assert.Equal(t, `"0"`, mustMarshalJSON(t, testmodels.OpaqueScalar{}))

	require.Len(t, scalar.ComponentPlans, 2)
	unit := scalar.ComponentPlans[0]
	assert.True(t, unit.IsHidden)
	assert.Equal(t, []string{"s", "m"}, unit.Examples)
	assert.ElementsMatch(t, []string{"s", "m"}, unit.Values)
	require.NotEmpty(t, unit.Rules)
	assert.Equal(t, []string{"scalar is set"}, unit.Rules[0].Conditions)
	value := scalar.ComponentPlans[1]
	require.Len(t, value.Rules, 2)
	assert.Equal(t, []string{"scalar is set", "unit is s"}, value.Rules[0].Conditions)
	assert.Equal(t, []string{"scalar is set", "unit is m"}, value.Rules[1].Conditions)

	var encoded struct {
		ComponentPlans []govy.PropertyPlan `json:"componentPlans"`
	}
	require.NoError(t, json.Unmarshal([]byte(mustMarshalJSON(t, scalar)), &encoded))
	assert.JSONEq(t, mustMarshalJSON(t, scalar.ComponentPlans), mustMarshalJSON(t, encoded.ComponentPlans))
}

func TestWithOpaqueType_RootComponentPlans(t *testing.T) {
	t.Run("value root", func(t *testing.T) {
		validator := opaqueScalarValidator()
		plan, err := govy.Plan(validator)
		require.NoError(t, err)

		doc, err := Generate(validator, WithOpaqueType[*testmodels.OpaqueScalar]("string"))
		require.NoError(t, err)
		assert.Equal(t, []string{"$"}, propertyPaths(doc))
		assertOpaqueScalarPlans(t, doc, plan, "$")
	})

	t.Run("pointer root", func(t *testing.T) {
		validator := govy.New(
			govy.ForPointer(govy.GetSelf[*testmodels.OpaqueScalar]()).Include(opaqueScalarValidator()),
		)
		plan, err := govy.Plan(validator)
		require.NoError(t, err)

		doc, err := Generate(validator, WithOpaqueType[testmodels.OpaqueScalar]("string"))
		require.NoError(t, err)
		assert.Equal(t, []string{"$"}, propertyPaths(doc))
		assertOpaqueScalarPlans(t, doc, plan, "$")
	})
}

func TestWithOpaqueType_UnregisteredComponentsRemainExcluded(t *testing.T) {
	doc, err := Generate(opaqueScalarsValidator())
	require.NoError(t, err)

	for _, property := range doc.Properties {
		assert.Empty(t, property.ComponentPlans)
	}
	scalar := requireProperty(t, doc, "$.scalar")
	assert.Equal(t, "struct", scalar.TypeInfo.Kind)
	assert.NotContains(t, propertyPaths(doc), "$.scalar.unit")
	assert.NotContains(t, propertyPaths(doc), "$.scalar.value")
	assert.NotContains(t, propertyPaths(doc), "$.secret")
}

func TestWithOpaqueType_CollectionComponentPlans(t *testing.T) {
	validator := opaqueScalarsValidator()
	plan, err := govy.Plan(validator)
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		option         GenerateOption
		componentPaths []string
	}{
		{
			name:   "slice",
			path:   "$.items",
			option: WithOpaqueType[[]testmodels.OpaqueScalar]("string"),
			componentPaths: []string{
				"$.items[*].unit", "$.items[*].value",
			},
		},
		{
			name:   "map",
			path:   "$.mapping",
			option: WithOpaqueType[map[testmodels.OpaqueScalar]testmodels.OpaqueScalar]("string"),
			componentPaths: []string{
				"$.mapping.*.unit", "$.mapping.*.value", "$.mapping.*~.unit", "$.mapping.*~.value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Generate(validator, WithOpaqueType[testmodels.OpaqueScalar]("string"), tt.option)
			require.NoError(t, err)
			property := requireProperty(t, doc, tt.path)
			assert.Equal(t, "string", property.TypeInfo.Kind)
			assert.Empty(t, property.ChildrenPaths)
			require.Len(t, property.ComponentPlans, len(tt.componentPaths))
			for i, path := range tt.componentPaths {
				assert.NotContains(t, propertyPaths(doc), path)
				assert.Equal(t, requirePlanProperty(t, plan, path), property.ComponentPlans[i])
			}
		})
	}
}

func TestWithOpaqueType_EscapedComponentPath(t *testing.T) {
	validator := govy.New(
		govy.For(testmodels.OpaqueScalar.Unit).
			WithPath(jsonpath.New().Name("unit.symbol")).Rules(rules.OneOf("s", "m")),
	)
	plan, err := govy.Plan(validator)
	require.NoError(t, err)

	doc, err := Generate(validator, WithOpaqueType[testmodels.OpaqueScalar]("string"))
	require.NoError(t, err)
	assert.Equal(t, []string{"$"}, propertyPaths(doc))
	property := requireProperty(t, doc, "$")
	require.Len(t, property.ComponentPlans, 1)
	assert.Equal(t, requirePlanProperty(t, plan, "$['unit.symbol']"), property.ComponentPlans[0])
}

func TestWithOpaqueType_FilteredPaths(t *testing.T) {
	validator := opaqueScalarsValidator()
	option := WithOpaqueType[testmodels.OpaqueScalar]("string")

	t.Run("opaque property", func(t *testing.T) {
		doc, err := Generate(validator, option, WithFilteredPaths("$.scalar"))
		require.NoError(t, err)
		assert.NotContains(t, propertyPaths(doc), "$.scalar")
	})

	t.Run("internal component", func(t *testing.T) {
		plan, err := govy.Plan(validator)
		require.NoError(t, err)
		doc, err := Generate(validator, option, WithFilteredPaths("$.scalar.unit"))
		require.NoError(t, err)
		assertOpaqueScalarPlans(t, doc, plan, "$.scalar")
	})
}

func opaqueScalarValidator() govy.Validator[testmodels.OpaqueScalar] {
	return govy.New(
		govy.For(testmodels.OpaqueScalar.Unit).
			WithName("unit").Required().HideValue().WithExamples("s", "m").
			Rules(rules.OneOf("s", "m").WithDetails("Suffix for the numeric value.").WithExamples("s")),
		govy.For(testmodels.OpaqueScalar.Value).
			WithName("value").WithExamples("1", "2").
			Rules(rules.GT(0).WithErrorCode("positive_value")).
			When(func(s testmodels.OpaqueScalar) bool { return s.Unit() == "s" }, govy.WhenDescription("unit is s")),
		govy.For(testmodels.OpaqueScalar.Value).
			WithName("value").
			Rules(rules.LTE(60)).
			When(func(s testmodels.OpaqueScalar) bool { return s.Unit() == "m" }, govy.WhenDescription("unit is m")),
	)
}

func opaqueScalarsValidator() govy.Validator[testmodels.StructWithOpaqueScalars] {
	scalar := opaqueScalarValidator()
	return govy.New(
		govy.For(func(s testmodels.StructWithOpaqueScalars) testmodels.OpaqueScalar { return s.Scalar }).
			WithName("scalar").OmitEmpty().WithExamples("5s").Include(scalar).
			When(func(s testmodels.StructWithOpaqueScalars) bool { return s.Scalar.Unit() != "" },
				govy.WhenDescription("scalar is set")),
		govy.For(func(s testmodels.StructWithOpaqueScalars) testmodels.OpaqueScalar { return s.Scalars }).
			WithName("scalars").Required().Include(scalar),
		govy.ForPointer(func(s testmodels.StructWithOpaqueScalars) *testmodels.OpaqueScalar { return s.Pointer }).
			WithName("pointer").Include(scalar),
		govy.For(func(s testmodels.StructWithOpaqueScalars) testmodels.NestedOpaqueScalar { return s.Nested }).
			WithName("nested").Include(govy.New(
			govy.For(func(s testmodels.NestedOpaqueScalar) testmodels.OpaqueScalar { return s.Scalar }).
				WithName("scalar").Include(scalar),
		)),
		govy.ForSlice(func(s testmodels.StructWithOpaqueScalars) []testmodels.OpaqueScalar { return s.Items }).
			WithName("items").IncludeForEach(scalar),
		govy.ForMap(func(s testmodels.StructWithOpaqueScalars) map[testmodels.OpaqueScalar]testmodels.OpaqueScalar {
			return s.Mapping
		}).WithName("mapping").IncludeForKeys(scalar).IncludeForValues(scalar),
		govy.For(func(s testmodels.StructWithOpaqueScalars) testmodels.OpaqueScalar { return s.Escaped }).
			WithPath(jsonpath.New().Name("scalar.value")).Include(scalar),
		govy.For(func(testmodels.StructWithOpaqueScalars) string { return "" }).
			WithName("secret").Required(),
	)
}

func assertOpaqueScalarPlans(t *testing.T, doc ObjectDoc, plan *govy.ValidatorPlan, path string) {
	t.Helper()
	property := requireProperty(t, doc, path)
	assert.Equal(t, "string", property.TypeInfo.Kind)
	assert.Empty(t, property.ChildrenPaths)
	require.Len(t, property.ComponentPlans, 2)
	for i, name := range []string{"unit", "value"} {
		componentPath := jsonpath.Parse(path).Name(name)
		assert.NotContains(t, propertyPaths(doc), componentPath.String())
		expected := requirePlanProperty(t, plan, componentPath.String())
		assert.Equal(t, expected, property.ComponentPlans[i])
	}
	for _, expected := range plan.Properties {
		if expected.Path.String() == path {
			parent := *expected
			parent.TypeInfo.Kind = "string"
			assert.Equal(t, parent, property.PropertyPlan)
			return
		}
	}
	assert.Empty(t, property.Rules)
	assert.Empty(t, property.Values)
	assert.Empty(t, property.Examples)
}

func requirePlanProperty(t *testing.T, plan *govy.ValidatorPlan, path string) govy.PropertyPlan {
	t.Helper()
	for _, property := range plan.Properties {
		if property.Path.String() == path {
			return *property
		}
	}
	require.FailNow(t, "validation property not found", "path: %s", path)
	return govy.PropertyPlan{}
}
