package govydoc

import (
	"fmt"
	"reflect"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/nobl9/govy/pkg/jsonpath"

	"github.com/nieomylnieja/govydoc/internal/godoc"
)

// ObjectDoc describes a Go type, its properties, and its validation documentation.
type ObjectDoc struct {
	Name       string        `json:"name"`
	Properties []PropertyDoc `json:"properties"`
	Examples   []Example     `json:"examples,omitempty,omitzero"`
	Doc        string        `json:"doc,omitempty"`
}

// Example describes a named usage example included in generated documentation.
type Example struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// PropertyDoc combines a govy property plan with its Go source documentation.
type PropertyDoc struct {
	govy.PropertyPlan
	// TypeDoc contains the documentation for the property's Go type.
	TypeDoc string `json:"typeDoc,omitempty"`
	// FieldDoc contains the documentation attached to the struct field.
	FieldDoc string `json:"fieldDoc,omitempty"`
	// DeprecatedDoc contains the text following a Deprecated marker.
	DeprecatedDoc string `json:"deprecatedDoc,omitempty"`
	// ChildrenPaths contains the JSON paths of the property's immediate children.
	ChildrenPaths []string `json:"childrenPaths,omitempty,omitzero"`
}

// GenerateOption configures [Generate].
type GenerateOption func(options generateOptions) generateOptions

type generateOptions struct {
	govyPlanOptions []govy.PlanOption
	filterPaths     []jsonpath.Path
}

// Generate returns documentation for the type handled by validator.
// It returns an error when source documentation or the govy validation plan cannot be generated.
func Generate[T any](validator govy.Validator[T], opts ...GenerateOption) (ObjectDoc, error) {
	typ := reflect.TypeFor[T]()

	options := generateOptions{}
	for _, opt := range opts {
		options = opt(options)
	}

	objectDoc := generateObjectDoc(typ)
	goDocParser, err := godoc.NewParser()
	if err != nil {
		return ObjectDoc{}, fmt.Errorf("failed to create Go documentation parser: %w", err)
	}
	goDoc, err := goDocParser.Parse(typ)
	if err != nil {
		return ObjectDoc{}, fmt.Errorf("failed to parse documentation for %s: %w", typ, err)
	}

	plan, err := govy.Plan(validator, options.govyPlanOptions...)
	if err != nil {
		return ObjectDoc{}, fmt.Errorf("failed to generate validation plan for %s: %w", typ, err)
	}
	objectDoc.extendWithValidationPlan(plan)

	mergeDocs(&objectDoc, goDoc)
	objectDoc = postProcessProperties(
		objectDoc,
		options.filterPaths,
		removeEnumDeclaration,
		extractDeprecatedInformation,
		removeTrailingWhitespace,
	)
	return objectDoc, nil
}

// GenerateGovyOptions returns an option that passes govyOptions to [govy.Plan].
func GenerateGovyOptions(govyOptions ...govy.PlanOption) GenerateOption {
	return func(options generateOptions) generateOptions {
		options.govyPlanOptions = append(options.govyPlanOptions, govyOptions...)
		return options
	}
}

// WithFilteredPaths returns an option that excludes the supplied JSON paths from generated documentation.
func WithFilteredPaths(paths ...string) GenerateOption {
	return func(options generateOptions) generateOptions {
		for _, path := range paths {
			options.filterPaths = append(options.filterPaths, jsonpath.Parse(path))
		}
		return options
	}
}

func (p PropertyDoc) key() string {
	if p.TypeInfo.Package == "" {
		return p.TypeInfo.Name
	}
	return p.TypeInfo.Package + "." + p.TypeInfo.Name
}

func mergeDocs(objectDoc *ObjectDoc, goDocs godoc.Docs) {
	for i, property := range objectDoc.Properties {
		if property.TypeInfo.Package == "" {
			continue
		}
		goDoc, found := goDocs[property.key()]
		if !found {
			continue
		}
		property.TypeDoc = goDoc.Doc
		for name, field := range goDoc.StructFields {
			fieldPath := property.Path.Name(name)
			for j, p := range objectDoc.Properties {
				if fieldPath.Equal(p.Path) {
					objectDoc.Properties[j].FieldDoc = field.Doc
					break
				}
			}
		}
		objectDoc.Properties[i] = property
	}
}

func (o *ObjectDoc) extendWithValidationPlan(plan *govy.ValidatorPlan) {
	o.Name = plan.Name
	for _, propPlan := range plan.Properties {
		for i, propDoc := range o.Properties {
			if !propPlan.Path.Equal(propDoc.Path) {
				continue
			}
			o.Properties[i] = PropertyDoc{
				PropertyPlan:  *propPlan,
				ChildrenPaths: propDoc.ChildrenPaths,
			}
			break
		}
	}
}
