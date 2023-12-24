package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

func deconstructSelector(node ast.Node) (string, bool) {
	identExpr, ok := node.(*ast.Ident)
	if ok {
		return identExpr.Name, true
	}
	selectorExpr, ok := node.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	prefix, ok := deconstructSelector(selectorExpr.X)
	if !ok {
		return "", false
	}
	return prefix + "." + selectorExpr.Sel.Name, true
}

func recognizeSafePattern(node ast.Node, typeInfo *types.Info) bool {
	return recognizeMapClearPattern(node) ||
		recognizePlatformDependentCode(node) ||
		recognizeConstantIfCondition(node, typeInfo) ||
		recognizeSafeAssignment(node, typeInfo) ||
		recognizeSafeDeclaration(node, typeInfo)
}

func recognizeSafeDeclaration(node ast.Node, typeInfo *types.Info) bool {
	declStmt, ok := node.(*ast.DeclStmt)
	if !ok {
		return false
	}
	genDecl := declStmt.Decl.(*ast.GenDecl)
	if genDecl.Tok != token.VAR {
		return false
	}
	for _, spec := range genDecl.Specs {
		for _, value := range spec.(*ast.ValueSpec).Values {
			if !recognizeSafeDeclarationRhs(value, typeInfo) {
				return false
			}
		}
	}
	return true
}

func recognizeSafeAssignment(node ast.Node, typeInfo *types.Info) bool {
	assignStmt, ok := node.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, rhs := range assignStmt.Rhs {
		if assignStmt.Tok == token.DEFINE && !recognizeSafeDeclarationRhs(rhs, typeInfo) {
			return false
		}
		if assignStmt.Tok != token.DEFINE && !recognizeSafeAssignmentRhs(rhs, typeInfo) {
			return false
		}
	}
	return true
}

func recognizeSafeDeclarationRhs(rhs ast.Expr, typeInfo *types.Info) bool {
	if identExpr, ok := rhs.(*ast.Ident); ok {
		return identExpr.Name == "false" || identExpr.Name == "true"
	}
	return recognizeSafeAssignmentRhs(rhs, typeInfo)
}

func recognizeSafeAssignmentRhs(rhs ast.Expr, typeInfo *types.Info) bool {
	if callExpr, ok := rhs.(*ast.CallExpr); ok {
		selector, _ := deconstructSelector(callExpr.Fun)
		if selector == "context.Background" {
			return true
		}
		exprTypeInfo := typeInfo.Types[callExpr.Fun].Type
		if exprTypeInfo == nil {
			return false
		}
		typeString := exprTypeInfo.String()
		if typeString == selector {
			return true
		}
		selectorTokens := strings.Split(selector, ".")
		typeStringTokens := strings.Split(typeString, ".")
		if typeStringTokens[len(typeStringTokens)-1] == selectorTokens[len(selectorTokens)-1] {
			return true
		}
	}
	return false
}

func recognizeConstantIfCondition(node ast.Node, typeInfo *types.Info) bool {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return false
	}
	return recognizeConstantFalse(ifStmt.Cond, typeInfo)
}

func recognizeConstantFalse(node ast.Expr, typeInfo *types.Info) bool {
	typeAndValue, ok := typeInfo.Types[node]
	if ok && typeAndValue.Value != nil && typeAndValue.Value.ExactString() == "false" {
		return true
	}
	binExpr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return binExpr.Op == token.LAND && (recognizeConstantFalse(binExpr.X, typeInfo) || recognizeConstantFalse(binExpr.Y, typeInfo))
}

func recognizePlatformDependentCode(node ast.Node) bool {
	if ifStmt, ok := node.(*ast.IfStmt); ok {
		return recognizePlatformDependentVarUsage(ifStmt.Cond)
	}
	if caseStmt, ok := node.(*ast.CaseClause); ok {
		for _, expr := range caseStmt.List {
			if recognizePlatformDependentVarUsage(expr) {
				return true
			}
		}
	}
	return false
}

func recognizePlatformDependentVarUsage(node ast.Node) bool {
	useGoos := false
	ast.Inspect(node, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		useGoos = useGoos || (ok && selector.Sel.Name == "GOOS")
		return true
	})
	return useGoos
}

func recognizeMapClearPattern(node ast.Node) bool {
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
	return equalExprs(rangeStmt.X, first) && equalExprs(rangeStmt.Key, second)
}
