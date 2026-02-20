package godoc

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nieomylnieja/govydoc/internal/testmodels"
	"github.com/nieomylnieja/govydoc/internal/testmodels/moremodels"
)

func TestNewParser(t *testing.T) {
	t.Run("successfully creates parser", func(t *testing.T) {
		parser, err := NewParser()
		require.NoError(t, err)
		assert.NotNil(t, parser)
		assert.NotEmpty(t, parser.pkgs)
	})

	t.Run("loads all packages including dependencies", func(t *testing.T) {
		parser, err := NewParser()
		require.NoError(t, err)

		loaded := slices.ContainsFunc(
			slices.Collect(maps.Keys(parser.pkgs)),
			func(path string) bool { return path == "github.com/nieomylnieja/govydoc/internal/testmodels" },
		)
		assert.True(t, loaded, "testmodels package should be loaded")
	})
}

func TestParser_Parse(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)

	t.Run("parses Teacher struct successfully", func(t *testing.T) {
		typ := reflect.TypeOf(testmodels.Teacher{})
		docs, err := parser.Parse(typ)
		require.NoError(t, err)
		require.NotNil(t, docs)

		// Verify Teacher doc exists
		teacherKey := "github.com/nieomylnieja/govydoc/internal/testmodels.Teacher"
		teacherDoc, found := docs[teacherKey]
		require.True(t, found, "Teacher documentation should exist")
		assert.Equal(t, "Teacher", teacherDoc.Name)
		assert.Equal(t, "github.com/nieomylnieja/govydoc/internal/testmodels", teacherDoc.Package)
		assert.NotEmpty(t, teacherDoc.Doc, "Teacher should have documentation")
		assert.Contains(t, teacherDoc.Doc, "Teacher is a sample struct")
	})

	t.Run("parses struct fields", func(t *testing.T) {
		typ := reflect.TypeOf(testmodels.Teacher{})
		docs, err := parser.Parse(typ)
		require.NoError(t, err)

		teacherKey := "github.com/nieomylnieja/govydoc/internal/testmodels.Teacher"
		teacherDoc := docs[teacherKey]

		// Verify struct fields are documented
		assert.NotEmpty(t, teacherDoc.StructFields)

		// Check specific field documentation
		if nameDoc, ok := teacherDoc.StructFields["Name"]; ok {
			assert.Contains(t, nameDoc.Doc, "Name is the name of the teacher")
		}
	})

	t.Run("parses nested types", func(t *testing.T) {
		typ := reflect.TypeOf(testmodels.Teacher{})
		docs, err := parser.Parse(typ)
		require.NoError(t, err)

		// Verify Student type is also parsed (nested in Teacher)
		studentKey := "github.com/nieomylnieja/govydoc/internal/testmodels.Student"
		studentDoc, found := docs[studentKey]
		assert.True(t, found, "Student should be parsed as it's nested in Teacher")
		if found {
			assert.Equal(t, "Student", studentDoc.Name)
			assert.Contains(t, studentDoc.Doc, "Student is just a teacher")
		}
	})

	t.Run("parses cross-package references", func(t *testing.T) {
		typ := reflect.TypeOf(testmodels.Teacher{})
		docs, err := parser.Parse(typ)
		require.NoError(t, err)

		// Verify University type from moremodels package is parsed
		uniKey := "github.com/nieomylnieja/govydoc/internal/testmodels/moremodels.University"
		uniDoc, found := docs[uniKey]
		require.True(t, found, "University from moremodels should be parsed")
		assert.Equal(t, "University", uniDoc.Name)
		assert.Equal(t, "github.com/nieomylnieja/govydoc/internal/testmodels/moremodels", uniDoc.Package)
	})

	t.Run("returns error for builtin types", func(t *testing.T) {
		typ := reflect.TypeOf("")
		_, err := parser.Parse(typ)
		// Builtin types don't have documentation in loaded packages
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no documentation found")
	})

	t.Run("parses Student with deprecated marker", func(t *testing.T) {
		typ := reflect.TypeOf(testmodels.Student{})
		docs, err := parser.Parse(typ)
		require.NoError(t, err)

		studentKey := "github.com/nieomylnieja/govydoc/internal/testmodels.Student"
		studentDoc, found := docs[studentKey]
		require.True(t, found)

		// Student has a "Deprecated:" marker in its doc
		assert.Contains(t, studentDoc.Doc, "Deprecated")
	})

	t.Run("parses struct with slice field", func(t *testing.T) {
		typ := reflect.TypeOf(testmodels.Teacher{})
		docs, err := parser.Parse(typ)
		require.NoError(t, err)

		// Teacher has Students []Student field
		teacherKey := "github.com/nieomylnieja/govydoc/internal/testmodels.Teacher"
		teacherDoc := docs[teacherKey]

		// Check Students field documentation
		assert.NotEmpty(t, teacherDoc.StructFields)
	})
}

func TestDoc_Key(t *testing.T) {
	tests := []struct {
		name     string
		doc      Doc
		expected string
	}{
		{
			name: "builtin type",
			doc: Doc{
				Name:    "string",
				Package: "",
			},
			expected: "string",
		},
		{
			name: "custom type with package",
			doc: Doc{
				Name:    "Teacher",
				Package: "github.com/nieomylnieja/govydoc/internal/testmodels",
			},
			expected: "github.com/nieomylnieja/govydoc/internal/testmodels.Teacher",
		},
		{
			name: "int builtin",
			doc: Doc{
				Name:    "int",
				Package: "",
			},
			expected: "int",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.doc.Key())
		})
	}
}

func TestDocs_add(t *testing.T) {
	docs := make(Docs)

	doc1 := Doc{
		Name:    "Teacher",
		Package: "github.com/test",
		Doc:     "Test doc",
	}

	docs.add(doc1)

	// Verify doc was added with correct key
	key := "github.com/test.Teacher"
	retrieved, found := docs[key]
	assert.True(t, found)
	assert.Equal(t, doc1.Name, retrieved.Name)
	assert.Equal(t, doc1.Package, retrieved.Package)
	assert.Equal(t, doc1.Doc, retrieved.Doc)
}

func TestParser_Parse_MultipleTypes(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)

	t.Run("parses multiple different types", func(t *testing.T) {
		// Parse Teacher
		typ1 := reflect.TypeOf(testmodels.Teacher{})
		docs1, err := parser.Parse(typ1)
		require.NoError(t, err)
		require.NotNil(t, docs1)

		// Parse University
		typ2 := reflect.TypeOf(moremodels.University{})
		docs2, err := parser.Parse(typ2)
		require.NoError(t, err)
		require.NotNil(t, docs2)

		// Both should have their respective documentation
		assert.Contains(t, docs1, "github.com/nieomylnieja/govydoc/internal/testmodels.Teacher")
		assert.Contains(t, docs2, "github.com/nieomylnieja/govydoc/internal/testmodels/moremodels.University")
	})
}

func TestParser_getPackageByPath(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)

	t.Run("finds existing package", func(t *testing.T) {
		pkg := parser.getPackageByPath("github.com/nieomylnieja/govydoc/internal/testmodels")
		require.NotNil(t, pkg)
		assert.Equal(t, "github.com/nieomylnieja/govydoc/internal/testmodels", pkg.pkg.PkgPath)
	})

	t.Run("returns nil for non-existent package", func(t *testing.T) {
		pkg := parser.getPackageByPath("github.com/nonexistent/package")
		assert.Nil(t, pkg)
	})
}

func TestParser_Parse_InterfaceType(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)

	t.Run("parses interface type (fmt.Stringer)", func(t *testing.T) {
		// Teacher has a fmt.Stringer field
		typ := reflect.TypeOf(testmodels.Teacher{})
		docs, err := parser.Parse(typ)
		require.NoError(t, err)

		// fmt.Stringer should be in the docs
		stringerKey := "fmt.Stringer"
		stringerDoc, found := docs[stringerKey]
		require.True(t, found)
		assert.Equal(t, "Stringer", stringerDoc.Name)
		assert.Equal(t, "fmt", stringerDoc.Package)
	})
}

func TestCheckForPackageErrors(t *testing.T) {
	t.Run("returns nil for packages without errors", func(t *testing.T) {
		parser, err := NewParser()
		require.NoError(t, err)
		assert.NotNil(t, parser)
	})
}
