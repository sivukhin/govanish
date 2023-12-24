package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

var simpleBuiltins = []string{
	"cap",
	"len",
	"complex",
	"imag",
	"real",
	"bool",
	"byte",
	"string",
	"complex128",
	"complex64",
	"float32",
	"float64",
	"int",
	"int16",
	"int32",
	"int64",
	"int8",
	"rune",
	"uint",
	"uint16",
	"uint32",
	"uint64",
	"uint8",
	"uintptr",
}

func checkNodeComplexity(node ast.Node) bool {
	complexFlow := false
	operations := 0
	ast.Inspect(node, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.ReturnStmt, *ast.ForStmt, *ast.RangeStmt, *ast.DeferStmt:
			complexFlow = true
		case *ast.CallExpr:
			if ident, ok := n.Fun.(*ast.Ident); ok {
				complexFlow = complexFlow || !slices.Contains(simpleBuiltins, ident.Name)
			} else {
				complexFlow = true
			}
		case *ast.UnaryExpr, *ast.BinaryExpr, *ast.IndexExpr, *ast.SliceExpr, *ast.StarExpr:
			operations += 1
		}
		return true
	})
	return complexFlow || operations >= 8
}

func reportIfVanished(
	fset *token.FileSet,
	assemblyLines map[string][]int,
	currentFunc string,
	start, end ast.Node,
) bool {
	startPosition, endPosition := fset.Position(start.Pos()), fset.Position(end.End())
	lines, ok := assemblyLines[startPosition.Filename]
	if !ok {
		return false
	}
	index, _ := slices.BinarySearch(lines, startPosition.Line)
	if index < len(lines) && lines[index] <= endPosition.Line {
		return false
	}
	snippet := ""
	f, err := os.OpenFile(startPosition.Filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	startLineEndPosition := fset.Position(start.End())
	_, _ = f.Seek(int64(startPosition.Offset), 0)
	buffer := make([]byte, startLineEndPosition.Offset-startPosition.Offset)
	_, _ = f.Read(buffer)
	snippet = string(buffer)

	log.Printf("it seems like your code vanished from compiled binary: func=[%v], file=[%v], line=[%v], snippet:\n\t%v", currentFunc, startPosition.Filename, startPosition.Line, snippet)
	return true
}

func isGenericFunc(funcDecl *ast.FuncDecl) bool {
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

func isPivot(node ast.Node) (body *ast.BlockStmt, ok bool) {
	switch n := node.(type) {
	case *ast.IfStmt:
		return n.Body, true
	case *ast.ForStmt:
		return n.Body, true
	case *ast.RangeStmt:
		return n.Body, true
	}
	return nil, false
}

func equalExprs(a, b ast.Expr) bool {
	aIdent, aOk := a.(*ast.Ident)
	bIdent, bOk := b.(*ast.Ident)
	if aOk && bOk {
		return aIdent.Name == bIdent.Name
	}
	aSelector, aOk := a.(*ast.SelectorExpr)
	bSelector, bOk := b.(*ast.SelectorExpr)
	if aOk && bOk {
		return aSelector.Sel.Name == bSelector.Sel.Name && equalExprs(aSelector.X, bSelector.X)
	}
	aStar, aOk := a.(*ast.StarExpr)
	bStar, bOk := b.(*ast.StarExpr)
	if aOk && bOk {
		return equalExprs(aStar.X, bStar.X)
	}
	return false
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	log.Printf("cwd: %v", cwd)

	cmd := exec.Command("go", "build", "-gcflags", "-S", "./...")
	log.Printf("ready to compile project for assembly inspection")
	go func() { cmd.Start() }()
	stdout, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	assemblyLines := make(map[string][]int)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		cwdIndex := strings.Index(line, cwd)
		if cwdIndex == -1 {
			continue
		}
		lineRefEnd := strings.Index(line[cwdIndex:], "\t")
		if lineRefEnd == -1 {
			panic(fmt.Errorf("unexpected line structure: %v", line))
		}
		fileRefEnd := strings.Index(line[cwdIndex:], ":")
		if fileRefEnd == -1 {
			panic(fmt.Errorf("unexpected line structure: %v", line))
		}
		fileRef := line[cwdIndex : cwdIndex+fileRefEnd]
		lineRef := line[cwdIndex+fileRefEnd+1 : cwdIndex+lineRefEnd-1]
		lineNumber, err := strconv.Atoi(strings.Trim(lineRef, ")]"))
		if err != nil {
			panic(fmt.Errorf("unexpected line structure: %v, err=%w", line, err))
		}
		assemblyLines[fileRef] = append(assemblyLines[fileRef], lineNumber)
	}
	log.Printf("assembly information were indexed, prepare it for queries")
	for fileName, lineNumbers := range assemblyLines {
		sort.Slice(lineNumbers, func(i, j int) bool { return lineNumbers[i] < lineNumbers[j] })
		deduplicated := make([]int, 0)
		for i := 0; i < len(lineNumbers); i++ {
			if i == 0 || lineNumbers[i] != lineNumbers[i-1] {
				deduplicated = append(deduplicated, lineNumbers[i])
			}
		}
		assemblyLines[fileName] = deduplicated
	}
	log.Printf("assembly information were prepared, ready to process AST")
	cfg := &packages.Config{
		Mode:  packages.NeedSyntax | packages.NeedFiles | packages.NeedTypes | packages.NeedTypesInfo,
		Tests: false,
	}

	project, err := packages.Load(cfg, "./...")
	if err != nil {
		panic(err)
	}
	for _, pkg := range project {
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			var currentFunc string
			var analyze func(node ast.Node) bool

			analyze = func(node ast.Node) bool {
				if funcDecl, ok := node.(*ast.FuncDecl); ok {
					currentFunc = funcDecl.Name.Name
					// don't process body of generic functions because code for them emitted
					return !isGenericFunc(funcDecl)
				}
				if recognizeSafePattern(node, pkg.TypesInfo) {
					return false
				}
				if _, pivot := isPivot(node); pivot {
					return true
				}
				blockStmt, ok := node.(*ast.BlockStmt)
				if !ok {
					return true
				}
				previous := -1
				for i := 0; i <= len(blockStmt.List); i++ {
					pivot := false
					if i == len(blockStmt.List) {
						pivot = true
					} else if _, ok := isPivot(blockStmt.List[i]); ok {
						pivot = true
					} else if recognizeSafePattern(blockStmt.List[i], pkg.TypesInfo) {
						pivot = true
					}
					if !pivot {
						continue
					}
					if checkNodeComplexity(&ast.BlockStmt{List: blockStmt.List[previous+1 : i]}) {
						reportIfVanished(pkg.Fset, assemblyLines, currentFunc, blockStmt.List[previous+1], blockStmt.List[i-1])
					}
					previous = i
				}
				return true
			}
			ast.Inspect(file, analyze)
		}
	}
}
