package main

import (
	"go/ast"
	"go/token"
	"strings"
)

func RecognizeSafeDeclaration(ctx GovanishContext, node ast.Node) bool {
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
			if !RecognizeSafeDeclarationRhs(ctx, value) {
				return false
			}
		}
	}
	return true
}

func RecognizeSafeAssignment(ctx GovanishContext, node ast.Node) bool {
	assignStmt, ok := node.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, rhs := range assignStmt.Rhs {
		if assignStmt.Tok == token.DEFINE && !RecognizeSafeDeclarationRhs(ctx, rhs) {
			return false
		}
		if assignStmt.Tok != token.DEFINE && !RecognizeSafeAssignmentRhs(ctx, rhs) {
			return false
		}
	}
	return true
}

const (
	False = "false"
	True  = "true"
)

func RecognizeSafeDeclarationRhs(ctx GovanishContext, rhs ast.Expr) bool {
	if identExpr, ok := rhs.(*ast.Ident); ok {
		return identExpr.Name == False || identExpr.Name == True
	}
	return RecognizeSafeAssignmentRhs(ctx, rhs)
}

var SimpleStructs = NewSet("context.Background", "context.TODO")

func RecognizeSafeAssignmentRhs(ctx GovanishContext, rhs ast.Expr) bool {
	// recognize type constructors (like a := T(b) where type T int64) or common structs with simple fields which efficiently inlined by compiler and legally vanished from assembly
	callExpr, ok := rhs.(*ast.CallExpr)
	if !ok {
		return false
	}
	expr := callExpr.Fun
	for {
		parenExpr, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = parenExpr.X
	}
	// recognize cast of pointers
	if _, ok := expr.(*ast.StarExpr); ok {
		return true
	}
	selector, _ := DeconstructSelector(expr)
	if SimpleStructs.Has(selector) {
		return true
	}
	exprTypeInfo := ctx.Pkg.TypesInfo.Types[expr].Type
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

func RecognizeConstantIfCondition(ctx GovanishContext, node ast.Node) bool {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return false
	}
	return RecognizeConstantFalse(ctx, ifStmt.Cond) || RecognizeConstantTrue(ctx, ifStmt.Cond)
}

func RecognizeConstantTrue(ctx GovanishContext, node ast.Expr) bool {
	typeAndValue, ok := ctx.Pkg.TypesInfo.Types[node]
	if ok && typeAndValue.Value != nil && typeAndValue.Value.ExactString() == True {
		return true
	}
	binExpr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return binExpr.Op == token.LOR && (RecognizeConstantTrue(ctx, binExpr.X) || RecognizeConstantTrue(ctx, binExpr.Y))
}

func RecognizeConstantFalse(ctx GovanishContext, node ast.Expr) bool {
	typeAndValue, ok := ctx.Pkg.TypesInfo.Types[node]
	if ok && typeAndValue.Value != nil && typeAndValue.Value.ExactString() == False {
		return true
	}
	binExpr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return binExpr.Op == token.LAND && (RecognizeConstantFalse(ctx, binExpr.X) || RecognizeConstantFalse(ctx, binExpr.Y))
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

func RecognizeDeterministicIfCondition(ctx GovanishContext, node ast.Node) bool {
	if ifStmt, ok := node.(*ast.IfStmt); ok && ifStmt.Init != nil {
		return recognizeDeterministicIfCondition(ctx, ifStmt.Init, ifStmt.Cond)
	}
	if block, ok := node.(*ast.BlockStmt); ok && len(block.List) == 2 {
		initStmt, okInit := block.List[0].(ast.Stmt)
		ifStmt, okIf := block.List[1].(*ast.IfStmt)
		if !okInit || !okIf {
			return false
		}
		if ifStmt.Init == nil {
			return recognizeDeterministicIfCondition(ctx, initStmt, ifStmt.Cond)
		}
	}
	return false
}

func recognizeDeterministicIfCondition(ctx GovanishContext, init ast.Stmt, condition ast.Expr) bool {
	assignStmt, ok := init.(*ast.AssignStmt)
	if !ok {
		return false
	}
	staticIdents := make(map[string]struct{})
	if len(assignStmt.Rhs) == 1 && len(assignStmt.Lhs) > 1 && recognizeDeterministicCall(ctx, assignStmt.Rhs[0]) {
		for _, lhs := range assignStmt.Lhs {
			if selector, ok := DeconstructSelector(lhs); ok {
				staticIdents[selector] = struct{}{}
			}
		}
	} else {
		for i, rhs := range assignStmt.Rhs {
			lhs := assignStmt.Lhs[i]
			if selector, ok := DeconstructSelector(lhs); ok && recognizeDeterministicCall(ctx, rhs) {
				staticIdents[selector] = struct{}{}
			}
		}
	}
	return recognizeDeterministicExpr(ctx, staticIdents, condition)
}

func recognizeDeterministicCall(ctx GovanishContext, node ast.Node) bool {
	callExpr, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := DeconstructSelector(callExpr.Fun)
	if !ok {
		return false
	}
	tokens := strings.Split(selector, ".")
	return ctx.FuncRegistry[tokens[len(tokens)-1]].DeterministicReturn
}

func recognizeDeterministicExpr(ctx GovanishContext, staticIdents map[string]struct{}, expr ast.Expr) bool {
	deterministic := true
	ast.Inspect(expr, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.CallExpr:
			deterministic = false
		case *ast.Ident:
			selector, _ := DeconstructSelector(n)
			if _, ok := staticIdents[selector]; ok {
				break
			}
			if typeAndValue, ok := ctx.Pkg.TypesInfo.Types[n]; ok && (typeAndValue.IsNil() || typeAndValue.Value != nil) {
				break
			}
			deterministic = false
		}
		return true
	})
	return deterministic
}
