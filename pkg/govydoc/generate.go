package govydoc

import (
	"reflect"

	"github.com/nobl9/govy/pkg/govy"
	"github.com/pkg/errors"

	"github.com/nieomylnieja/govydoc/internal/godoc"
)

type ObjectDoc struct {
	Name       string        `json:"name"`
	Properties []PropertyDoc `json:"properties"`
	Examples   []Example     `json:"examples,omitempty"`
	Doc        string        `json:"doc,omitempty"`
}

type Example struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type PropertyDoc struct {
	govy.PropertyPlan
	// TypeDoc holds the documentation for the given type.
	// For instance, if property is an object of type X,
	// the TypeDoc will contain the X's documentation.
	TypeDoc string `json:"typeDoc,omitempty"`
	// FieldDoc holds the inline documentation which was provided on the struct field level.
	FieldDoc string `json:"fieldDoc,omitempty"`
	// DeprecatedDoc holds property's "Deprecated:" comment contents.
	DeprecatedDoc string   `json:"deprecatedDoc,omitempty"`
	ChildrenPaths []string `json:"childrenPaths,omitempty"`
}

func (p PropertyDoc) key() string {
	if p.TypeInfo.Package == "" {
		return p.TypeInfo.Name
	}
	return p.TypeInfo.Package + "." + p.TypeInfo.Name
}

// generateOptions contains options for configuring the behavior of the [Generate] function.
type generateOptions struct {
	govyPlanOptions []govy.PlanOption
	filterPaths     []string
}

type GenerateOption func(options generateOptions) generateOptions

// GenerateGovyOptions allows you to provide [govy.PlanOption] to the internally called [govy.Plan].
func GenerateGovyOptions(govyOptions ...govy.PlanOption) GenerateOption {
	return func(options generateOptions) generateOptions {
		options.govyPlanOptions = append(options.govyPlanOptions, govyOptions...)
		return options
	}
}

// WithFilteredPaths specifies property paths that should be excluded from the generated documentation.
// Paths use JSONPath notation (e.g., "$.organization", "$.metadata.internal").
func WithFilteredPaths(paths ...string) GenerateOption {
	return func(options generateOptions) generateOptions {
		options.filterPaths = append(options.filterPaths, paths...)
		return options
	}
}

func Generate[T any](validator govy.Validator[T], opts ...GenerateOption) (ObjectDoc, error) {
	typ := reflect.TypeOf(*new(T))

	options := generateOptions{}
	for _, opt := range opts {
		options = opt(options)
	}

	objectDoc := generateObjectDoc(typ)
	goDocParser, err := godoc.NewParser()
	if err != nil {
		return ObjectDoc{}, err
	}
	goDoc, err := goDocParser.Parse(typ)
	if err != nil {
		return ObjectDoc{}, err
	}

	plan, err := govy.Plan(validator, options.govyPlanOptions...)
	if err != nil {
		var t T
		return ObjectDoc{}, errors.Wrapf(err, "failed to generate validation plan for %T", t)
	}
	objectDoc.extendWithValidationPlan(plan)

	mergeDocs(&objectDoc, goDoc)
	return postProcessProperties(objectDoc, options.filterPaths,
		removeEnumDeclaration,
		extractDeprecatedInformation,
		removeTrailingWhitespace,
	), nil
}

func mergeDocs(objectDoc *ObjectDoc, goDocs godoc.Docs) {
	for i, property := range objectDoc.Properties {
		// Builtin type.
		if property.TypeInfo.Package == "" {
			continue
		}
		goDoc, found := goDocs[property.key()]
		if !found {
			continue
		}
		property.TypeDoc = goDoc.Doc
		for name, field := range goDoc.StructFields {
			fieldPath := property.Path + "." + name
			for j, p := range objectDoc.Properties {
				if fieldPath == p.Path {
					objectDoc.Properties[j].FieldDoc = field.Doc
					break
				}
			}
		}
		objectDoc.Properties[i] = property
	}
}

// extendWithValidationPlan extends [ObjectDoc.Properties] with [govy.ValidatorPlan] results.
func (o *ObjectDoc) extendWithValidationPlan(plan *govy.ValidatorPlan) {
	o.Name = plan.Name
	for _, propPlan := range plan.Properties {
		for i, propDoc := range o.Properties {
			if propPlan.Path != propDoc.Path {
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
