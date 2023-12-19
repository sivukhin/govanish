package main

import (
	"bufio"
	"fmt"
	"go/ast"
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
func api() *E              { return nil }

func use() {
	var err error = api()
	if err == nil {
		panic("this is impossible")
	}
}

func checkSubtreeComplexity(node ast.Node) bool {
	isComplex := false
	ast.Inspect(node, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.CallExpr:
			isComplex = true
		case *ast.ReturnStmt:
			isComplex = true
		}
		return true
	})
	return isComplex
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
			ast.Inspect(file, func(node ast.Node) bool {
				if funcDecl, ok := node.(*ast.FuncDecl); ok {
					currentFunc = funcDecl.Name.Name
					return true
				}
				_, ok := node.(*ast.IfStmt)
				if !ok {
					return true
				}
				if !checkSubtreeComplexity(node) {
					return false
				}
				start, end := pkg.Fset.Position(node.Pos()), pkg.Fset.Position(node.End())
				startLine, endLine := start.Line, end.Line
				lines, ok := assemblyLines[start.Filename]
				if !ok {
					return true
				}
				position, _ := slices.BinarySearch(lines, startLine)
				if position >= len(lines) || lines[position] > endLine {
					log.Printf("it seems like your code vanished from compiled binary: func=[%v], file=[%v], line=[%v]", currentFunc, start.Filename, start.Line)
				}
				return true
			})
		}
	}
}
