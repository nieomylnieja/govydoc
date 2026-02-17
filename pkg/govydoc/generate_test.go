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
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}
