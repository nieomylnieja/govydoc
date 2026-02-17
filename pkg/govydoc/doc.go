// Package govydoc generates comprehensive documentation for Go types validated with the govy library.
//
// It combines three sources of information:
//  1. Validation rules from govy validators
//  2. Go source code documentation (godoc comments)
//  3. Type structure information via reflection
//
// The resulting ObjectDoc contains property paths, type information, validation rules,
// and extracted documentation in a unified JSON-serializable format.
//
// # Basic Usage
//
// Given a struct with govy validation:
//
//	type Teacher struct {
//	    Name  string `json:"name"`
//	    Age   int    `json:"age"`
//	    Email string `json:"email"`
//	}
//
//	func teacherValidator() govy.Validator[Teacher] {
//	    return govy.New(
//	        govy.For(func(t Teacher) string { return t.Name }).
//	            Rules(validation.StringNotEmpty()),
//	        govy.For(func(t Teacher) int { return t.Age }).
//	            Rules(validation.IntRange(18, 100)),
//	    ).WithName("Teacher")
//	}
//
// Generate documentation:
//
//	doc, err := govydoc.Generate(teacherValidator())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The resulting ObjectDoc includes:
//   - Property paths (e.g., "$.name", "$.age")
//   - Type information for each property
//   - Validation rules and requirements
//   - Godoc comments from the source code
//   - Nested property relationships
//
// # Configuration Options
//
// Use GenerateOption functions to customize behavior:
//
//	doc, err := govydoc.Generate(
//	    validator,
//	    govydoc.WithFilteredPaths("$.internalField"),
//	    govydoc.GenerateGovyOptions(govy.WithExampleProvider(...)),
//	)
//
// WithFilteredPaths excludes specified property paths from documentation.
// GenerateGovyOptions passes options to the internal govy.Plan call.
//
// # Output Format
//
// ObjectDoc is JSON-serializable and contains:
//
//   - Name: The type name
//   - Properties: Array of PropertyDoc with path, type, validation rules, and documentation
//   - Examples: Optional usage examples
//   - Doc: Type-level documentation from godoc comments
//
// Each PropertyDoc includes:
//
//   - Path: JSONPath notation (e.g., "$.address.city")
//   - TypeInfo: Go type information (name, kind, package)
//   - Rules: Validation rules from govy
//   - TypeDoc: Documentation for the property's type
//   - FieldDoc: Inline documentation from the struct field
//   - DeprecatedDoc: Contents of "Deprecated:" comments
//   - ChildrenPaths: Paths of immediate nested properties
package govydoc
