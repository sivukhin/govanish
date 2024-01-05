package main

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

func RecognizeSafeDeclaration(pkg *packages.Package, node ast.Node) bool {
	declStmt, ok := node.(*ast.DeclStmt)
	if !ok {
		return false
	}
	genDecl := declStmt.Decl.(*ast.GenDecl)
	if genDecl.Tok == token.CONST {
		return true
	}
	if genDecl.Tok != token.VAR {
		return false
	}
	for _, spec := range genDecl.Specs {
		for _, value := range spec.(*ast.ValueSpec).Values {
			if !RecognizeSafeDeclarationRhs(pkg, value) {
				return false
			}
		}
	}
	return true
}

func RecognizeSafeAssignment(pkg *packages.Package, node ast.Node) bool {
	assignStmt, ok := node.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, rhs := range assignStmt.Rhs {
		if assignStmt.Tok == token.DEFINE && !RecognizeSafeDeclarationRhs(pkg, rhs) {
			return false
		}
		if assignStmt.Tok != token.DEFINE && !RecognizeSafeAssignmentRhs(pkg, rhs) {
			return false
		}
	}
	return true
}

const (
	False = "false"
	True  = "true"
)

func RecognizeSafeDeclarationRhs(pkg *packages.Package, rhs ast.Expr) bool {
	if identExpr, ok := rhs.(*ast.Ident); ok {
		return identExpr.Name == False || identExpr.Name == True
	}
	return RecognizeSafeAssignmentRhs(pkg, rhs)
}

var SimpleStructs = NewSet("context.Background")

func RecognizeSafeAssignmentRhs(pkg *packages.Package, rhs ast.Expr) bool {
	// recognize type constructors (like a := T(b) where type T int64) or common structs with simple fields which efficiently inlined by compiler and legally vanished from assembly
	callExpr, ok := rhs.(*ast.CallExpr)
	if !ok {
		return false
	}
	if parenExpr, ok := callExpr.Fun.(*ast.ParenExpr); ok {
		// recognize cast of pointers
		if _, ok := parenExpr.X.(*ast.StarExpr); ok {
			return true
		}
		return false
	}
	selector, _ := DeconstructSelector(callExpr.Fun)
	if SimpleStructs.Has(selector) {
		return true
	}
	exprTypeInfo := pkg.TypesInfo.Types[callExpr.Fun].Type
	if exprTypeInfo == nil {
		return false
	}
	typeString := exprTypeInfo.String()
	if typeString == selector {
		return true
	}
	// simple heuristic - we detect type constructor if last token in selector matches the last selector of FQN name of the type
	selectorTokens := strings.Split(selector, ".")
	typeStringTokens := strings.Split(typeString, ".")
	if typeStringTokens[len(typeStringTokens)-1] == selectorTokens[len(selectorTokens)-1] {
		return true
	}
	return false
}

func RecognizeConstantIfCondition(pkg *packages.Package, node ast.Node) bool {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return false
	}
	return RecognizeConstantFalse(pkg, ifStmt.Cond) || RecognizeConstantTrue(pkg, ifStmt.Cond)
}

func RecognizeConstantTrue(pkg *packages.Package, node ast.Expr) bool {
	typeAndValue, ok := pkg.TypesInfo.Types[node]
	if ok && typeAndValue.Value != nil && typeAndValue.Value.ExactString() == True {
		return true
	}
	binExpr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return binExpr.Op == token.LOR && (RecognizeConstantTrue(pkg, binExpr.X) || RecognizeConstantTrue(pkg, binExpr.Y))
}

func RecognizeConstantFalse(pkg *packages.Package, node ast.Expr) bool {
	typeAndValue, ok := pkg.TypesInfo.Types[node]
	if ok && typeAndValue.Value != nil && typeAndValue.Value.ExactString() == False {
		return true
	}
	binExpr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return binExpr.Op == token.LAND && (RecognizeConstantFalse(pkg, binExpr.X) || RecognizeConstantFalse(pkg, binExpr.Y))
}

var PlatformDependentSelectors = NewSet("runtime.GOOS", "runtime.GOARCH", "filepath.Separator", "filepath.ToSlash", "filepath.FromSlash", "os.PathSeparator")

func RecognizePlatformDependentCode(node ast.Node) bool {
	// ignore functions with platform dependent code inside
	if _, ok := node.(*ast.FuncDecl); !ok {
		return false
	}
	platformDependent := false
	ast.Inspect(node, func(node ast.Node) bool {
		selector, ok := DeconstructSelector(node)
		platformDependent = platformDependent || (ok && PlatformDependentSelectors.Has(selector))
		return true
	})
	return platformDependent
}

func RecognizeMapClearPattern(node ast.Node) bool {
	/*
		recognize map clear intent with range loop which compiled to the single runtime.mapclear() call
		for k := range m {
			delete(m, k)
		}
	*/
	rangeStmt, ok := node.(*ast.RangeStmt)
	if !ok {
		return false
	}
	body := rangeStmt.Body.List
	if len(body) != 1 {
		return false
	}
	expr, ok := body[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	funcName, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	if funcName.Name != "delete" {
		return false
	}
	if len(call.Args) != 2 {
		return false
	}
	first, second := call.Args[0], call.Args[1]
	return EqualExprs(rangeStmt.X, first) && EqualExprs(rangeStmt.Key, second)
}
