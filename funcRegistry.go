package main

import (
	"go/ast"
	"log"

	"golang.org/x/tools/go/packages"
)

type (
	FuncProps    struct{ DeterministicReturn bool }
	FuncRegistry map[string]FuncProps
)

func analyzeFunc(pkg *packages.Package, funcDecl *ast.FuncDecl) FuncProps {
	staticReturns := 0
	dynamicReturns := 0
	ast.Inspect(funcDecl, func(node ast.Node) bool {
		returnStmt, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		static, dynamic := 0, 0
		for _, result := range returnStmt.Results {
			typeAndValue, ok := pkg.TypesInfo.Types[result]
			if ok && (typeAndValue.Value != nil || typeAndValue.IsNil()) {
				static++
			} else {
				dynamic++
			}
		}
		if dynamic == 0 {
			staticReturns++
		} else {
			dynamicReturns++
		}
		return true
	})
	return FuncProps{DeterministicReturn: staticReturns <= 1 && dynamicReturns == 0}
}

func (r *FuncRegistry) fillFuncRegistryFromPkg(pkg *packages.Package, visitedPkgs map[string]struct{}) {
	if _, ok := visitedPkgs[pkg.ID]; ok {
		return
	}
	visitedPkgs[pkg.ID] = struct{}{}

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if funcDecl, ok := node.(*ast.FuncDecl); ok {
				(*r)[funcDecl.Name.Name] = analyzeFunc(pkg, funcDecl)
				return false
			}
			return true
		})
	}

	for _, importPkg := range pkg.Imports {
		r.fillFuncRegistryFromPkg(importPkg, visitedPkgs)
	}
}

func CreateFuncRegistry(pkgs []*packages.Package) FuncRegistry {
	visitedPkgs := make(map[string]struct{})
	funcRegistry := make(FuncRegistry)
	for _, pkg := range pkgs {
		funcRegistry.fillFuncRegistryFromPkg(pkg, visitedPkgs)
	}
	log.Printf("built func registry: %v entries", len(funcRegistry))
	return funcRegistry
}
