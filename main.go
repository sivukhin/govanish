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

type E struct{ Desc string }

func (e *E) Error() string { return e.Desc }
func api(n int) *E {
	if n == 0 {
		return nil
	}
	return &E{Desc: "error"}
}

func use(n int) {
	var err error = api(n)
	if err != nil {
		panic("this is impossible")
	}
	panic(fmt.Sprintf("this is possible: %v", err))
}

var simpleBuiltins = []string{
	"make",
	"cap",
	"len",
	"complex",
	"imag",
	"max",
	"min",
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
	isComplex := false
	ast.Inspect(node, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.ReturnStmt, *ast.ForStmt, *ast.RangeStmt, *ast.DeferStmt:
			isComplex = true
		case *ast.CallExpr:
			if ident, ok := n.Fun.(*ast.Ident); ok {
				isComplex = isComplex || !slices.Contains(simpleBuiltins, ident.Name)
			} else {
				isComplex = false
			}
		case *ast.BinaryExpr:
			isComplex = isComplex || n.Op == token.ARROW
		}
		return true
	})
	return isComplex
}

func reportIfVanished(
	fset *token.FileSet,
	assemblyLines map[string][]int,
	currentFunc string,
	start, end token.Pos,
) bool {
	startPosition, endPosition := fset.Position(start), fset.Position(end)
	lines, ok := assemblyLines[startPosition.Filename]
	if !ok {
		return false
	}
	index, _ := slices.BinarySearch(lines, startPosition.Line)
	if index < len(lines) && lines[index] <= endPosition.Line {
		return false
	}
	log.Printf("it seems like your code vanished from compiled binary: func=[%v], file=[%v], line=[%v]", currentFunc, startPosition.Filename, startPosition.Line)
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
		lineNumber, err := strconv.Atoi(lineRef)
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
		Mode:  packages.NeedSyntax | packages.NeedFiles | packages.NeedTypes,
		Tests: true,
	}

	project, err := packages.Load(cfg, "./...")
	if err != nil {
		panic(err)
	}
	for _, pkg := range project {
		for _, file := range pkg.Syntax {
			if strings.Contains(pkg.Fset.Position(file.Pos()).Filename, "generated") {
				continue
			}
			var currentFunc string
			var analyze func(node ast.Node) bool

			analyze = func(node ast.Node) bool {
				if funcDecl, ok := node.(*ast.FuncDecl); ok {
					if isGenericFunc(funcDecl) {
						return false
					}
					currentFunc = funcDecl.Name.Name
					return true
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
					} else {
						switch blockStmt.List[i].(type) {
						case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.BlockStmt:
							pivot = true
						}
					}
					if pivot {
						if checkNodeComplexity(&ast.BlockStmt{List: blockStmt.List[previous+1 : i]}) {
							reportIfVanished(pkg.Fset, assemblyLines, currentFunc, blockStmt.List[previous+1].Pos(), blockStmt.List[i-1].End())
						}
						previous = i
					}
				}
				return true
			}
			ast.Inspect(file, analyze)
		}
	}
}
