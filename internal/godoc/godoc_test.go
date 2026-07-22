package godoc

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"

	"github.com/nieomylnieja/govydoc/internal/testmodels"
	"github.com/nieomylnieja/govydoc/internal/testmodels/moremodels"
)

func TestNewParser(t *testing.T) {
	parser, err := NewParser()

	require.NoError(t, err)
	require.NotNil(t, parser)
	assert.NotEmpty(t, parser.pkgs)
	assert.Contains(t, parser.pkgs, testModelsPackage)
}

func TestParser_Parse(t *testing.T) {
	parser := newTestParser(t)
	docs, err := parser.Parse(reflect.TypeFor[testmodels.Teacher]())
	require.NoError(t, err)

	t.Run("type documentation", func(t *testing.T) {
		teacherDoc, found := docs[testModelsPackage+".Teacher"]
		require.True(t, found)
		assert.Equal(t, "Teacher", teacherDoc.Name)
		assert.Equal(t, testModelsPackage, teacherDoc.Package)
		assert.Contains(t, teacherDoc.Doc, "Teacher is a sample struct")
	})

	t.Run("struct field documentation", func(t *testing.T) {
		teacherDoc, found := docs[testModelsPackage+".Teacher"]
		require.True(t, found)
		nameDoc, found := teacherDoc.StructFields["name"]
		require.True(t, found)
		assert.Contains(t, nameDoc.Doc, "Name is the name of the teacher")
		assert.Contains(t, teacherDoc.StructFields, "students")
	})

	t.Run("nested type", func(t *testing.T) {
		studentDoc, found := docs[testModelsPackage+".Student"]
		require.True(t, found)
		assert.Equal(t, "Student", studentDoc.Name)
		assert.Contains(t, studentDoc.Doc, "Student is just a teacher")
	})

	t.Run("cross-package type", func(t *testing.T) {
		universityDoc, found := docs[moreModelsPackage+".University"]
		require.True(t, found)
		assert.Equal(t, "University", universityDoc.Name)
		assert.Equal(t, moreModelsPackage, universityDoc.Package)
	})

	t.Run("interface type", func(t *testing.T) {
		stringerDoc, found := docs["fmt.Stringer"]
		require.True(t, found)
		assert.Equal(t, "Stringer", stringerDoc.Name)
		assert.Equal(t, "fmt", stringerDoc.Package)
	})

	t.Run("deprecated marker", func(t *testing.T) {
		studentsField, found := reflect.TypeFor[testmodels.Teacher]().FieldByName("Students")
		require.True(t, found)
		studentDocs, err := parser.Parse(studentsField.Type.Elem())
		require.NoError(t, err)

		studentDoc, found := studentDocs[testModelsPackage+".Student"]
		require.True(t, found)
		assert.Contains(t, studentDoc.Doc, "Deprecated")
	})

	t.Run("nested pointer and slice type", func(t *testing.T) {
		typ := reflect.PointerTo(reflect.SliceOf(reflect.PointerTo(reflect.TypeFor[testmodels.Teacher]())))
		nestedDocs, err := parser.Parse(typ)
		require.NoError(t, err)
		assert.Contains(t, nestedDocs, testModelsPackage+".Teacher")
	})

	t.Run("built-in type", func(t *testing.T) {
		_, err := parser.Parse(reflect.TypeFor[string]())
		require.ErrorContains(t, err, "no documentation found")
	})

	t.Run("nil type", func(t *testing.T) {
		_, err := parser.Parse(nil)
		require.EqualError(t, err, "type cannot be nil")
	})
}

func TestParser_ParseMultipleTypes(t *testing.T) {
	parser := newTestParser(t)

	teacherDocs, err := parser.Parse(reflect.TypeFor[testmodels.Teacher]())
	require.NoError(t, err)
	universityDocs, err := parser.Parse(reflect.TypeFor[moremodels.University]())
	require.NoError(t, err)

	assert.Contains(t, teacherDocs, testModelsPackage+".Teacher")
	assert.Contains(t, universityDocs, moreModelsPackage+".University")
}

func TestDoc_Key(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		doc      Doc
		expected string
	}{
		"built-in type": {
			doc:      Doc{Name: "string"},
			expected: "string",
		},
		"custom type": {
			doc:      Doc{Name: "Teacher", Package: testModelsPackage},
			expected: testModelsPackage + ".Teacher",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, test.doc.Key())
		})
	}
}

func TestDocs_add(t *testing.T) {
	t.Parallel()

	docs := make(Docs)
	doc := Doc{Name: "Teacher", Package: "github.com/test", Doc: "Test doc"}

	docs.add(doc)

	retrieved, found := docs["github.com/test.Teacher"]
	require.True(t, found)
	assert.Equal(t, doc, retrieved)
}

func Test_checkForPackageErrors(t *testing.T) {
	t.Parallel()

	t.Run("no errors", func(t *testing.T) {
		t.Parallel()
		err := checkForPackageErrors([]*packages.Package{{PkgPath: "example.com/valid"}})
		require.NoError(t, err)
	})

	t.Run("one error", func(t *testing.T) {
		t.Parallel()
		err := checkForPackageErrors([]*packages.Package{{
			PkgPath: "example.com/broken",
			Errors:  []packages.Error{{Msg: "invalid source"}},
		}})
		require.ErrorContains(t, err, "package example.com/broken has reported an error")
		require.ErrorContains(t, err, "invalid source")
	})

	t.Run("multiple errors", func(t *testing.T) {
		t.Parallel()
		err := checkForPackageErrors([]*packages.Package{{
			PkgPath: "example.com/broken",
			Errors: []packages.Error{
				{Msg: "first error"},
				{Msg: "second error"},
			},
		}})
		require.ErrorContains(t, err, "encountered 2 errors while loading packages")
		require.ErrorContains(t, err, "first error")
		require.ErrorContains(t, err, "second error")
	})
}

func Test_embeddedFieldName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		expr     ast.Expr
		expected string
	}{
		"identifier": {
			expr:     ast.NewIdent("Local"),
			expected: "Local",
		},
		"qualified identifier": {
			expr: &ast.SelectorExpr{
				X:   ast.NewIdent("otherpkg"),
				Sel: ast.NewIdent("Remote"),
			},
			expected: "Remote",
		},
		"pointer to qualified identifier": {
			expr: &ast.StarExpr{X: &ast.SelectorExpr{
				X:   ast.NewIdent("otherpkg"),
				Sel: ast.NewIdent("Remote"),
			}},
			expected: "Remote",
		},
		"unsupported expression": {
			expr: &ast.ArrayType{Elt: ast.NewIdent("string")},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, embeddedFieldName(test.expr))
		})
	}
}

const (
	testModelsPackage = "github.com/nieomylnieja/govydoc/internal/testmodels"
	moreModelsPackage = testModelsPackage + "/moremodels"
)

func newTestParser(t *testing.T) *Parser {
	t.Helper()
	parser, err := NewParser()
	require.NoError(t, err)
	return parser
}
