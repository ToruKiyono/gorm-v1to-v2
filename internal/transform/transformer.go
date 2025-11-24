package transform

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// Transformer migrates GORM v1 code to GORM v2.
type Transformer struct {
	Verbose bool
}

// Transform walks the given path, updating all Go files (excluding vendor) in place.
func (t *Transformer) Transform(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		if err := t.TransformFile(path); err != nil {
			return fmt.Errorf("transform %s: %w", path, err)
		}
		return nil
	})
}

// TransformFile rewrites a single Go file.
func (t *Transformer) TransformFile(path string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	ctx := newFileContext(file, fset)

	astutil.Apply(file, ctx.pre, ctx.post)

	ctx.removeLogModeStatements()
	ctx.ensureImports()

	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(buf.String()), 0o644)
}

type fileContext struct {
	file          *ast.File
	fset          *token.FileSet
	driverImports map[string]struct{}
	gormAliases   map[string]struct{}
}

func newFileContext(file *ast.File, fset *token.FileSet) *fileContext {
	ctx := &fileContext{
		file:          file,
		fset:          fset,
		driverImports: map[string]struct{}{},
		gormAliases:   map[string]struct{}{`gorm`: {}},
	}

	for _, im := range file.Imports {
		importPath := strings.Trim(im.Path.Value, "\"")
		if importPath == "github.com/jinzhu/gorm" || importPath == "gorm.io/gorm" {
			if im.Name != nil && im.Name.Name != "" {
				ctx.gormAliases[im.Name.Name] = struct{}{}
			} else {
				ctx.gormAliases[`gorm`] = struct{}{}
			}
		}
	}

	return ctx
}

func (c *fileContext) isGormIdent(id *ast.Ident) bool {
	if id == nil {
		return false
	}
	_, ok := c.gormAliases[id.Name]
	return ok
}

func (c *fileContext) pre(cursor *astutil.Cursor) bool {
	node := cursor.Node()
	switch n := node.(type) {
	case *ast.CallExpr:
		c.rewriteGormOpen(cursor, n)
		c.rewriteLogMode(cursor, n)
	}
	return true
}

func (c *fileContext) post(*astutil.Cursor) bool {
	return true
}

func (c *fileContext) rewriteGormOpen(cursor *astutil.Cursor, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Open" {
		return
	}

	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok || pkgIdent.Name == "" {
		return
	}

	if !c.isGormIdent(pkgIdent) {
		return
	}

	if len(call.Args) < 2 {
		return
	}

	dialectLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || dialectLit.Kind != token.STRING {
		return
	}

	dialect := strings.Trim(dialectLit.Value, "\"")
	driverImport, driverName := driverForDialect(dialect)
	if driverImport == "" {
		return
	}

	dsn := call.Args[1]

	newCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{X: pkgIdent, Sel: ast.NewIdent("Open")},
		Args: []ast.Expr{
			&ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(driverName), Sel: ast.NewIdent("Open")}, Args: []ast.Expr{dsn}},
			&ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{Type: &ast.SelectorExpr{X: pkgIdent, Sel: ast.NewIdent("Config")}}},
		},
	}

	cursor.Replace(newCall)
	c.driverImports[driverImport] = struct{}{}
}

func (c *fileContext) rewriteLogMode(cursor *astutil.Cursor, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "LogMode" {
		return
	}

	receiver := sel.X
	cursor.Replace(receiver)
}

func (c *fileContext) removeLogModeStatements() {
	ast.Inspect(c.file, func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}

		var filtered []ast.Stmt
		for _, stmt := range block.List {
			es, ok := stmt.(*ast.ExprStmt)
			if !ok {
				filtered = append(filtered, stmt)
				continue
			}

			call, ok := es.X.(*ast.CallExpr)
			if !ok {
				filtered = append(filtered, stmt)
				continue
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "LogMode" {
				filtered = append(filtered, stmt)
				continue
			}
			// Drop the LogMode statement entirely.
		}

		block.List = filtered
		return true
	})
}

func (c *fileContext) ensureImports() {
	astutil.AddImport(c.fset, c.file, "gorm.io/gorm")

	for imp := range c.driverImports {
		astutil.AddImport(c.fset, c.file, imp)
	}

	astutil.DeleteImport(c.fset, c.file, "github.com/jinzhu/gorm")
}

func driverForDialect(dialect string) (importPath, name string) {
	switch strings.ToLower(dialect) {
	case "mysql":
		return "gorm.io/driver/mysql", "mysql"
	case "postgres", "postgresql":
		return "gorm.io/driver/postgres", "postgres"
	case "sqlite", "sqlite3":
		return "gorm.io/driver/sqlite", "sqlite"
	case "mssql", "sqlserver":
		return "gorm.io/driver/sqlserver", "sqlserver"
	default:
		return "", ""
	}
}
