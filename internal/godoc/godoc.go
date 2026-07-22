// Package godoc extracts documentation for Go types from a module's source packages.
package godoc

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc/comment"
	"go/types"
	"maps"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"

	"github.com/nieomylnieja/govydoc/internal/modroot"
)

const docLinkBaseURL = "https://pkg.go.dev"

// Docs maps fully qualified Go type names to their documentation.
type Docs map[string]Doc

// Doc describes a Go type and, for structs, its fields.
type Doc struct {
	Name         string
	Package      string
	Doc          string
	StructFields Docs
}

// Parser extracts Go documentation from the packages in a module.
type Parser struct {
	pkgs map[string]*goPackage
}

type goPackage struct {
	pkg           *packages.Package
	commentParser *comment.Parser
}

// NewParser returns a parser initialized with every package reachable from the current Go module.
func NewParser() (*Parser, error) {
	root, err := modroot.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to find module root: %w", err)
	}

	config := &packages.Config{
		Dir: root,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(config, "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}
	if err = checkForPackageErrors(pkgs); err != nil {
		return nil, err
	}

	parser := &Parser{pkgs: make(map[string]*goPackage, len(pkgs))}
	parser.collectAllPackages(pkgs)
	return parser, nil
}

// Key returns the type's package-qualified name, or its name for built-in types.
func (d Doc) Key() string {
	if d.Package == "" {
		return d.Name
	}
	return d.Package + "." + d.Name
}

// Parse returns documentation for goType and the named types reachable through its fields.
func (p *Parser) Parse(goType reflect.Type) (Docs, error) {
	if goType == nil {
		return nil, errors.New("type cannot be nil")
	}

	m := make(Docs)
	if _, err := p.parse(goType, m); err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("no documentation found for type %s", goType)
	}
	return m, nil
}

func (d Docs) add(doc Doc) {
	d[doc.Key()] = doc
}

func (p *Parser) parse(goType reflect.Type, docs Docs) (*Doc, error) {
	for goType.Kind() == reflect.Pointer || goType.Kind() == reflect.Slice {
		goType = goType.Elem()
	}

	name := goType.Name()
	pkgPath := goType.PkgPath()
	typeDoc := Doc{
		Name:    name,
		Package: pkgPath,
	}

	if pkgPath == "" {
		return &typeDoc, nil
	}

	pkg, decl, err := p.getTypeDeclarationInfo(pkgPath, name)
	if err != nil {
		return nil, err
	}
	typeDoc.Doc = docCommentToMarkdown(pkg.commentParser, pkg.pkg.PkgPath, decl.Doc.Text())

	if goType.Kind() != reflect.Struct {
		docs.add(typeDoc)
		return &typeDoc, nil
	}

	if err := p.parseStructFields(goType, &typeDoc, pkg, decl, docs); err != nil {
		return nil, err
	}

	docs.add(typeDoc)
	return &typeDoc, nil
}

func (p *Parser) getTypeDeclarationInfo(pkgPath, name string) (*goPackage, *ast.GenDecl, error) {
	pkg := p.pkgs[pkgPath]
	if pkg == nil {
		return nil, nil, fmt.Errorf("could not find %s package for type %s", pkgPath, name)
	}
	if pkg.commentParser == nil {
		pkg.commentParser = p.newCommentParserForPackage(pkg.pkg)
	}

	decl, err := findTypeDeclaration(pkg, name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find %s declaration in package %s: %w", name, pkgPath, err)
	}

	return pkg, decl, nil
}

func (p *Parser) parseStructFields(
	goType reflect.Type,
	typeDoc *Doc,
	pkg *goPackage,
	decl *ast.GenDecl,
	docs Docs,
) error {
	structType, err := extractStructType(decl, typeDoc.Name)
	if err != nil {
		return err
	}

	typeDoc.StructFields = make(Docs, goType.NumField())
	astFieldsByName := buildASTFieldMap(structType)

	for field := range goType.Fields() {
		if err := p.parseStructField(field, typeDoc, pkg, astFieldsByName, docs); err != nil {
			return err
		}
	}

	return nil
}

func extractStructType(decl *ast.GenDecl, name string) (*ast.StructType, error) {
	if len(decl.Specs) == 0 {
		return nil, fmt.Errorf("no specs found in declaration for %s", name)
	}
	typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("expected *ast.TypeSpec for %s, got %T", name, decl.Specs[0])
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("failed to parse %s struct type, expected ast.StructType", name)
	}
	return structType, nil
}

// buildASTFieldMap maps reflected field names to their AST declarations.
// A single AST field can declare multiple reflected fields, so their indexes do not reliably correspond.
func buildASTFieldMap(structType *ast.StructType) map[string]*ast.Field {
	astFieldsByName := make(map[string]*ast.Field)
	for _, astField := range structType.Fields.List {
		for _, fieldName := range astField.Names {
			astFieldsByName[fieldName.Name] = astField
		}
		if len(astField.Names) != 0 {
			continue
		}
		if name := embeddedFieldName(astField.Type); name != "" {
			astFieldsByName[name] = astField
		}
	}
	return astFieldsByName
}

func embeddedFieldName(expr ast.Expr) string {
	switch typ := expr.(type) {
	case *ast.Ident:
		return typ.Name
	case *ast.SelectorExpr:
		return typ.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(typ.X)
	default:
		return ""
	}
}

func (p *Parser) parseStructField(
	goTypeField reflect.StructField,
	typeDoc *Doc,
	pkg *goPackage,
	astFieldsByName map[string]*ast.Field,
	docs Docs,
) error {
	fieldDoc, err := p.parse(goTypeField.Type, docs)
	if err != nil {
		return fmt.Errorf("failed to parse %s struct field %s: %w", typeDoc.Name, goTypeField.Name, err)
	}

	fieldName := getStructFieldName(goTypeField)
	if fieldName == "" {
		return nil
	}

	if astField, ok := astFieldsByName[goTypeField.Name]; ok {
		fieldDoc.Doc = docCommentToMarkdown(pkg.commentParser, pkg.pkg.PkgPath, astField.Doc.Text())
	}

	typeDoc.StructFields[fieldName] = *fieldDoc
	return nil
}

func findTypeDeclaration(pkg *goPackage, name string) (*ast.GenDecl, error) {
	obj := pkg.pkg.Types.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("%s.%s not found", pkg.pkg.Types.Path(), name)
	}
	for _, file := range pkg.pkg.Syntax {
		pos := obj.Pos()
		if file.FileStart > pos || pos >= file.FileEnd {
			continue
		}
		path, _ := astutil.PathEnclosingInterval(file, pos, pos)
		for _, n := range path {
			if n, ok := n.(*ast.GenDecl); ok {
				return n, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find %s.%s declaration", pkg.pkg.Name, name)
}

func docCommentToMarkdown(parser *comment.Parser, pkg, text string) string {
	if text == "" {
		return ""
	}
	typeDoc := parser.Parse(text)
	printer := comment.Printer{
		DocLinkURL: func(link *comment.DocLink) string {
			if link.ImportPath == "" {
				link.ImportPath = pkg
			}
			return link.DefaultURL(docLinkBaseURL)
		},
	}
	return string(printer.Markdown(typeDoc))
}

func (p *Parser) newCommentParserForPackage(currentPackage *packages.Package) *comment.Parser {
	return &comment.Parser{
		LookupPackage: func(name string) (importPath string, ok bool) {
			for _, pkg := range p.pkgs {
				if pkg.pkg.Name == name {
					return pkg.pkg.PkgPath, true
				}
			}
			return "", false
		},
		LookupSym: func(recv, name string) (ok bool) {
			if recv == "" {
				return currentPackage.Types.Scope().Lookup(name) != nil
			}
			obj := currentPackage.Types.Scope().Lookup(recv)
			if obj == nil {
				return false
			}
			switch u := obj.Type().Underlying().(type) {
			case *types.Struct:
				for field := range u.Fields() {
					if field.Name() == name {
						return true
					}
				}
				return false
			default:
				return false
			}
		},
	}
}

func (p *Parser) collectAllPackages(pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		if _, exists := p.pkgs[pkg.PkgPath]; exists {
			continue
		}
		p.pkgs[pkg.PkgPath] = &goPackage{pkg: pkg}
		p.collectAllPackages(slices.Collect(maps.Values(pkg.Imports)))
	}
}

func checkForPackageErrors(pkgs []*packages.Package) error {
	var errs []error
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		for _, pkgErr := range pkg.Errors {
			errs = append(errs, fmt.Errorf("package %s has reported an error: %w", pkg.PkgPath, pkgErr))
		}
		mod := pkg.Module
		if mod != nil && mod.Error != nil {
			errs = append(errs, fmt.Errorf("module %s has error: %s", mod.Path, mod.Error.Err))
		}
		return true
	}, nil)

	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return fmt.Errorf("encountered %d errors while loading packages: %w", len(errs), errors.Join(errs...))
	}
}

func getStructFieldName(field reflect.StructField) string {
	if !field.IsExported() {
		return ""
	}
	tagName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if tagName == "" {
		return field.Name
	}
	if tagName == "-" {
		return ""
	}
	return tagName
}
