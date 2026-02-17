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
	root := pathutils.FindModuleRoot()
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
	if pkgPath == "" {
		// Builtin type, no need to parse.
		return &typeDoc, nil
	}

	// Find the package and package-level object.
	pkg := p.getPackageByPath(pkgPath)
	if pkg == nil {
		return nil, errors.Errorf("could not find %s package for type %s", pkgPath, name)
	}
	if pkg.commentParser == nil {
		pkg.commentParser = p.newCommentParserForPackage(pkg.pkg)
	}

	decl, err := p.findTypeDeclaration(pkg, name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find %s declaration in %s pkg", name, pkgPath)
	}
	typeDoc.Doc = p.docCommentToMarkdown(pkg.commentParser, pkg.pkg.PkgPath, decl.Doc.Text())

	// We're done for anything other than a struct.
	if goType.Kind() != reflect.Struct {
		docs.add(typeDoc)
		return &typeDoc, nil
	}

	structType, ok := decl.Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)
	if !ok {
		return nil, errors.Errorf("failed to parse %s struct type, expected ast.StructType", name)
	}
	typeDoc.StructFields = make(Docs, goType.NumField())
	// If the type is a struct, we need to go over its fields.
	for i, astField := range structType.Fields.List {
		goTypeField := goType.Field(i)
		fieldDoc, err := p.parse(goTypeField.Type, docs)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %s struct field %s", name, goTypeField.Name)
		}
		fieldName := getStructFieldName(goTypeField)
		if fieldName == "" {
			continue // Skip unexported fields.
		}
		fieldDoc.Doc = p.docCommentToMarkdown(pkg.commentParser, pkg.pkg.PkgPath, astField.Doc.Text())
		typeDoc.StructFields[fieldName] = *fieldDoc
	}
	docs.add(typeDoc)
	return &typeDoc, nil
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

func checkForPackageErrors(pkgs []*packages.Package) (err error) {
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		for _, err = range pkg.Errors {
			err = errors.Wrapf(err, "package %s has reported an error", pkg.PkgPath)
			return false
		}
		mod := pkg.Module
		if mod != nil && mod.Error != nil {
			err = errors.New(mod.Error.Err)
			return false
		}
		return true
	}, nil)
	return err
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
