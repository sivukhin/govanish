package main

import (
	"go/ast"
	"log"
	"os"

	"golang.org/x/tools/go/packages"
)

type VanishedInfo struct {
	Pkg      *packages.Package
	FuncName string
	Start    ast.Node
	End      ast.Node
}

type AnalysisPolicy interface {
	ShouldSkip(pkg *packages.Package, node ast.Node) bool
	IsControlFlowPivot(node ast.Node) bool
	CheckComplexity(pkg *packages.Package, node ast.Node) bool
	ReportVanished(info VanishedInfo)
}

type GovanishAnalysisPolicy struct{}

var Govanish AnalysisPolicy = GovanishAnalysisPolicy{}

func (g GovanishAnalysisPolicy) ShouldSkip(pkg *packages.Package, node ast.Node) bool {
	if funcDecl, ok := node.(*ast.FuncDecl); ok && IsGenericFunc(funcDecl) {
		return true
	}
	if _, ok := node.(*ast.FuncLit); ok {
		return true
	}
	return RecognizeMapClearPattern(node) ||
		RecognizeConstantIfCondition(pkg, node) ||
		RecognizeSafeAssignment(pkg, node) ||
		RecognizeSafeDeclaration(pkg, node) ||
		RecognizePlatformDependentCode(node)
}

func (g GovanishAnalysisPolicy) IsControlFlowPivot(node ast.Node) bool {
	switch node.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
		return true
	}
	return false
}

var SimpleBuiltins = NewSet("make", "cap", "len", "complex", "imag", "real", "bool", "byte", "string", "complex128", "complex64", "float32", "float64", "int", "int16", "int32", "int64", "int8", "rune", "uint", "uint16", "uint32", "uint64", "uint8", "uintptr")

func (g GovanishAnalysisPolicy) CheckComplexity(pkg *packages.Package, node ast.Node) bool {
	// compiler can optimize some statements to the sequence of CMOV commands in which case some lines can be removed from assembly info but they will be still there
	complexFlow := false
	operations := 0
	ast.Inspect(node, func(node ast.Node) bool {
		if expr, ok := node.(ast.Expr); ok {
			if typeAndValue, ok := pkg.TypesInfo.Types[expr]; ok && typeAndValue.Value != nil {
				return false
			}
		}
		switch n := node.(type) {
		case *ast.ReturnStmt, *ast.ForStmt, *ast.RangeStmt, *ast.DeferStmt:
			complexFlow = true
		case *ast.CallExpr:
			if ident, ok := n.Fun.(*ast.Ident); ok {
				complexFlow = complexFlow || !SimpleBuiltins.Has(ident.Name)
			} else {
				complexFlow = true
			}
		case *ast.UnaryExpr:
			if _, ok := n.X.(*ast.BasicLit); !ok {
				operations += 1
			}
		case *ast.BinaryExpr, *ast.IndexExpr:
			operations += 1
		}
		return true
	})
	return complexFlow || operations >= 2
}

func (i VanishedInfo) Filename() string { return i.Pkg.Fset.Position(i.Start.Pos()).Filename }
func (i VanishedInfo) StartLine() int   { return i.Pkg.Fset.Position(i.Start.Pos()).Line }
func (i VanishedInfo) EndLine() int     { return i.Pkg.Fset.Position(i.End.Pos()).Line }
func (i VanishedInfo) StartLineOffsets() (start, end int) {
	startPos := i.Pkg.Fset.Position(i.Start.Pos())
	endPos := i.Pkg.Fset.Position(i.Start.End())
	return startPos.Offset, endPos.Offset
}

func (g GovanishAnalysisPolicy) ReportVanished(info VanishedInfo) {
	snippet := ""
	f, err := os.OpenFile(info.Filename(), os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	start, end := info.StartLineOffsets()
	_, _ = f.Seek(int64(start), 0)
	buffer := make([]byte, end-start)
	_, _ = f.Read(buffer)
	snippet = string(buffer)

	log.Printf(
		"it seems like your code vanished from compiled binary: func=[%v], file=[%v], lines=[%v-%v], snippet:\n\t%v",
		info.FuncName,
		info.Filename(),
		info.StartLine(),
		info.EndLine(),
		snippet,
	)
}
