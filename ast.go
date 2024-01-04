package main

import (
	"go/ast"
)

func IsGenericFunc(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.TypeParams != nil {
		return true
	}
	if funcDecl.Recv != nil {
		generic := false
		for _, recv := range funcDecl.Recv.List {
			target := recv.Type
			for {
				if star, ok := target.(*ast.StarExpr); ok {
					target = star.X
				} else {
					break
				}
			}
			if _, ok := target.(*ast.IndexListExpr); ok {
				generic = true
			}
			if _, ok := target.(*ast.IndexExpr); ok {
				generic = true
			}
		}
		return generic
	}
	return false
}

func EqualExprs(a, b ast.Expr) bool {
	aIdent, aOk := a.(*ast.Ident)
	bIdent, bOk := b.(*ast.Ident)
	if aOk && bOk {
		return aIdent.Name == bIdent.Name
	}
	aSelector, aOk := a.(*ast.SelectorExpr)
	bSelector, bOk := b.(*ast.SelectorExpr)
	if aOk && bOk {
		return aSelector.Sel.Name == bSelector.Sel.Name && EqualExprs(aSelector.X, bSelector.X)
	}
	aStar, aOk := a.(*ast.StarExpr)
	bStar, bOk := b.(*ast.StarExpr)
	if aOk && bOk {
		return EqualExprs(aStar.X, bStar.X)
	}
	aIndex, aOk := a.(*ast.IndexExpr)
	bIndex, bOk := b.(*ast.IndexExpr)
	if aOk && bOk {
		return EqualExprs(aIndex.X, bIndex.X) && EqualExprs(aIndex.Index, bIndex.Index)
	}
	aLit, aOk := a.(*ast.BasicLit)
	bLit, bOk := b.(*ast.BasicLit)
	if aOk && bOk {
		return aLit.Value == bLit.Value && aLit.Kind == bLit.Kind
	}
	return false
}

func DeconstructSelector(node ast.Node) (string, bool) {
	identExpr, ok := node.(*ast.Ident)
	if ok {
		return identExpr.Name, true
	}
	selectorExpr, ok := node.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	prefix, ok := DeconstructSelector(selectorExpr.X)
	if !ok {
		return "", false
	}
	return prefix + "." + selectorExpr.Sel.Name, true
}
