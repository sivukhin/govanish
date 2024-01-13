package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

func MustGenExpr(expr string) (*token.FileSet, ast.Expr) {
	fset, stmt := MustGenStatements(expr)
	return fset, stmt[0].(*ast.ExprStmt).X
}

func MustGenStatements(statements string) (*token.FileSet, []ast.Stmt) {
	fset, funcAst := MustGenFunc(fmt.Sprintf("func main() {\n%v\n}", statements))
	return fset, funcAst.Body.List
}

func MustGenFunc(src string) (*token.FileSet, *ast.FuncDecl) {
	fset, fileAst := MustGenSrc(fmt.Sprintf("package main\n%v", src))
	return fset, fileAst.Decls[0].(*ast.FuncDecl)
}

func MustGenSrc(src string) (*token.FileSet, *ast.File) {
	fset := token.NewFileSet()
	fileAst, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		panic(fmt.Errorf("src parsing failed: %w", err))
	}
	return fset, fileAst
}

func MustGenMod(src string) (string, func(), error) {
	dir, err := os.MkdirTemp(".", "test-*")
	if err != nil {
		return "", nil, err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", nil, err
	}
	f, err := os.Create(path.Join(dir, "main.go"))
	if err != nil {
		return "", nil, err
	}
	_, _ = f.WriteString(src)
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

func MustExtractFunc(pkgs []*packages.Package) *ast.FuncDecl {
	var result *ast.FuncDecl
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(node ast.Node) bool {
				if funcDecl, ok := node.(*ast.FuncDecl); ok && result == nil {
					result = funcDecl
				}
				return true
			})
		}
	}
	if result == nil {
		panic(fmt.Errorf("unable to find function"))
	}
	return result
}
