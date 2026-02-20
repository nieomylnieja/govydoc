package godoc

import (
	"go/ast"
	"go/doc/comment"
	"go/types"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"

	"github.com/nieomylnieja/govydoc/internal/pathutils"
)

type Docs map[string]Doc

func (d Docs) add(doc Doc) {
	d[doc.Key()] = doc
}

type Doc struct {
	Name         string
	Package      string
	Doc          string
	StructFields Docs
}

func (d Doc) Key() string {
	if d.Package == "" {
		return d.Name
	}
	return d.Package + "." + d.Name
}

func NewParser() (*Parser, error) {
	root, err := pathutils.FindModuleRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find module root")
	}
	// Load complete type information for the specified packages,
	// along with type-annotated syntax.
	conf := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(conf, root+"/...")
	if err != nil {
		return nil, errors.Wrap(err, "failed to load packages")
	}
	if err = checkForPackageErrors(pkgs); err != nil {
		return nil, err
	}

	parser := &Parser{pkgs: make(map[string]*goPackage, len(pkgs))}
	parser.collectAllPackages(pkgs)
	return parser, nil
}

type Parser struct {
	pkgs map[string]*goPackage
}

type goPackage struct {
	pkg           *packages.Package
	commentParser *comment.Parser
}

func (p *Parser) Parse(goType reflect.Type) (Docs, error) {
	m := make(Docs)
	if _, err := p.parse(goType, m); err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, errors.Errorf("no documentation found for type %s", goType)
	}
	return m, nil
}

func (p *Parser) parse(goType reflect.Type, docs Docs) (*Doc, error) {
	// nolint:exhaustive // Only handle pointer and slice kinds; other kinds fall through
	switch goType.Kind() {
	case reflect.Pointer, reflect.Slice:
		goType = goType.Elem()
	}

	name := goType.Name()
	pkgPath := goType.PkgPath()
	typeDoc := Doc{
		Name:    name,
		Package: pkgPath,
	}

	// Builtin types don't need further parsing
	if pkgPath == "" {
		return &typeDoc, nil
	}

	// Get package and parse type declaration
	pkg, decl, err := p.getTypeDeclarationInfo(pkgPath, name)
	if err != nil {
		return nil, err
	}
	typeDoc.Doc = p.docCommentToMarkdown(pkg.commentParser, pkg.pkg.PkgPath, decl.Doc.Text())

	// Non-struct types are done here
	if goType.Kind() != reflect.Struct {
		docs.add(typeDoc)
		return &typeDoc, nil
	}

	// Parse struct fields
	if err := p.parseStructFields(goType, &typeDoc, pkg, decl, docs); err != nil {
		return nil, err
	}

	docs.add(typeDoc)
	return &typeDoc, nil
}

// getTypeDeclarationInfo retrieves the package and AST declaration for a type
func (p *Parser) getTypeDeclarationInfo(pkgPath, name string) (*goPackage, *ast.GenDecl, error) {
	pkg := p.getPackageByPath(pkgPath)
	if pkg == nil {
		return nil, nil, errors.Errorf("could not find %s package for type %s", pkgPath, name)
	}
	if pkg.commentParser == nil {
		pkg.commentParser = p.newCommentParserForPackage(pkg.pkg)
	}

	decl, err := p.findTypeDeclaration(pkg, name)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find %s declaration in %s pkg", name, pkgPath)
	}

	return pkg, decl, nil
}

// parseStructFields parses all fields of a struct type
func (p *Parser) parseStructFields(
	goType reflect.Type,
	typeDoc *Doc,
	pkg *goPackage,
	decl *ast.GenDecl,
	docs Docs,
) error {
	structType, err := p.extractStructType(decl, typeDoc.Name)
	if err != nil {
		return err
	}

	typeDoc.StructFields = make(Docs, goType.NumField())
	astFieldsByName := p.buildASTFieldMap(structType)

	// Parse each field
	for i := 0; i < goType.NumField(); i++ {
		if err := p.parseStructField(goType.Field(i), typeDoc, pkg, astFieldsByName, docs); err != nil {
			return err
		}
	}

	return nil
}

// extractStructType extracts and validates the struct type from AST declaration
func (p *Parser) extractStructType(decl *ast.GenDecl, name string) (*ast.StructType, error) {
	if len(decl.Specs) == 0 {
		return nil, errors.Errorf("no specs found in declaration for %s", name)
	}
	typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return nil, errors.Errorf("expected *ast.TypeSpec for %s, got %T", name, decl.Specs[0])
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, errors.Errorf("failed to parse %s struct type, expected ast.StructType", name)
	}
	return structType, nil
}

// buildASTFieldMap creates a map from field names to AST fields for reliable matching.
// We cannot rely on index correspondence because:
// 1. AST fields with multiple names (e.g., "A, B int") create one ast.Field but multiple reflect fields
// 2. Embedded fields may have different ordering
func (p *Parser) buildASTFieldMap(structType *ast.StructType) map[string]*ast.Field {
	astFieldsByName := make(map[string]*ast.Field)
	for _, astField := range structType.Fields.List {
		// Regular fields with names
		for _, fieldName := range astField.Names {
			astFieldsByName[fieldName.Name] = astField
		}
		// Handle embedded fields (no names)
		if len(astField.Names) == 0 {
			p.addEmbeddedFieldToMap(astField, astFieldsByName)
		}
	}
	return astFieldsByName
}

// addEmbeddedFieldToMap extracts the type name from an embedded field and adds it to the map
func (p *Parser) addEmbeddedFieldToMap(astField *ast.Field, astFieldsByName map[string]*ast.Field) {
	if astField.Type == nil {
		return
	}

	switch typ := astField.Type.(type) {
	case *ast.Ident:
		astFieldsByName[typ.Name] = astField
	case *ast.SelectorExpr:
		if ident, ok := typ.X.(*ast.Ident); ok {
			astFieldsByName[ident.Name] = astField
		}
	case *ast.StarExpr:
		// Handle *Embedded case
		if ident, ok := typ.X.(*ast.Ident); ok {
			astFieldsByName[ident.Name] = astField
		}
	}
}

// parseStructField parses a single struct field
func (p *Parser) parseStructField(goTypeField reflect.StructField, typeDoc *Doc, pkg *goPackage,
	astFieldsByName map[string]*ast.Field, docs Docs,
) error {
	fieldDoc, err := p.parse(goTypeField.Type, docs)
	if err != nil {
		return errors.Wrapf(err, "failed to parse %s struct field %s", typeDoc.Name, goTypeField.Name)
	}

	fieldName := getStructFieldName(goTypeField)
	if fieldName == "" {
		return nil // Skip unexported fields
	}

	// Look up the corresponding AST field by name and extract doc comment
	if astField, ok := astFieldsByName[goTypeField.Name]; ok {
		fieldDoc.Doc = p.docCommentToMarkdown(pkg.commentParser, pkg.pkg.PkgPath, astField.Doc.Text())
	}

	typeDoc.StructFields[fieldName] = *fieldDoc
	return nil
}

// findTypeDeclaration finds the ast.GenDecl for the given type declaration, specified by name.
func (p *Parser) findTypeDeclaration(pkg *goPackage, name string) (*ast.GenDecl, error) {
	obj := pkg.pkg.Types.Scope().Lookup(name)
	if obj == nil {
		return nil, errors.Errorf("%s.%s not found", pkg.pkg.Types.Path(), name)
	}
	for _, file := range pkg.pkg.Syntax {
		pos := obj.Pos()
		if file.FileStart > pos || pos >= file.FileEnd {
			continue // not in this file
		}
		path, _ := astutil.PathEnclosingInterval(file, pos, pos)
		for _, n := range path {
			if n, ok := n.(*ast.GenDecl); ok {
				return n, nil
			}
		}
	}
	return nil, errors.Errorf("could not find %s.%s declaration", pkg.pkg.Name, name)
}

const docLinkBaseURL = "https://pkg.go.dev"

func (p *Parser) docCommentToMarkdown(parser *comment.Parser, pkg, text string) string {
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

func (p *Parser) getPackageByPath(pkgPath string) *goPackage {
	for path, pkg := range p.pkgs {
		if path == pkgPath {
			return pkg
		}
	}
	return nil
}

// collectAllPackages recursively adds all packages and their imports to the parser's map.
func (p *Parser) collectAllPackages(pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		if _, exists := p.pkgs[pkg.PkgPath]; exists {
			continue
		}
		p.pkgs[pkg.PkgPath] = &goPackage{pkg: pkg}
		if len(pkg.Imports) > 0 {
			p.collectAllPackages(slices.Collect(maps.Values(pkg.Imports)))
		}
	}
}

func checkForPackageErrors(pkgs []*packages.Package) error {
	var errs []error
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		for _, pkgErr := range pkg.Errors {
			errs = append(errs, errors.Wrapf(pkgErr, "package %s has reported an error", pkg.PkgPath))
		}
		mod := pkg.Module
		if mod != nil && mod.Error != nil {
			errs = append(errs, errors.Wrapf(errors.New(mod.Error.Err), "module %s has error", mod.Path))
		}
		return true // Continue visiting all packages
	}, nil)

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	// Format multiple errors clearly
	var msg strings.Builder
	msg.WriteString(errors.Errorf("encountered %d errors while loading packages", len(errs)).Error())
	for i, err := range errs {
		msg.WriteString(errors.Errorf("\n  %d. %v", i+1, err).Error())
	}
	return errors.New(msg.String())
}

func getStructFieldName(field reflect.StructField) string {
	if !field.IsExported() {
		return ""
	}
	tagValues := strings.Split(field.Tag.Get("json"), ",")
	if len(tagValues) == 0 {
		return field.Name
	}
	tagName := tagValues[0]
	if tagName == "" {
		return field.Name
	}
	if tagName == "-" {
		return ""
	}
	return tagName
}
