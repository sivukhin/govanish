package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"log"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

type GovanishContext struct {
	Pkg           *packages.Package
	AssemblyLines AssemblyLines
	FuncRegistry  FuncRegistry
}

func LoadPackage(dir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedSyntax | packages.NeedFiles | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Tests: false,
		Dir:   dir,
	}

	return packages.Load(cfg, "./...")
}

type AssemblyLines map[string][]int

func (assemblyLines AssemblyLines) Normalize() {
	log.Printf("ready to normalize assembly lines (size %v)", len(assemblyLines))
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
}

func ParseAssemblyOutput(path string, scanner *bufio.Scanner) AssemblyLines {
	log.Printf("ready to parse assembly output")
	assemblyLines := make(AssemblyLines)
	for scanner.Scan() {
		line := scanner.Text()
		cwdIndex := strings.Index(line, path)
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

	assemblyLines.Normalize()
	return assemblyLines
}

func AnalyzeModuleAssembly(path string) (AssemblyLines, error) {
	log.Printf("ready to compile project at path '%v' for assembly inspection", path)
	cmd := exec.Command("go", "build", "-C", path, "-gcflags", "-S", "./...")
	errs := make(chan error)
	go func() {
		defer close(errs)
		if err := cmd.Start(); err != nil {
			errs <- err
			return
		}
		if err := cmd.Wait(); err != nil {
			errs <- err
			return
		}
	}()
	stdout, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	assemblyLines := ParseAssemblyOutput(path, bufio.NewScanner(stdout))
	for err := range errs {
		return assemblyLines, err
	}
	return assemblyLines, nil
}

func IsVanished(pkg *packages.Package, assemblyLines AssemblyLines, start, end ast.Node) bool {
	startPosition, endPosition := pkg.Fset.Position(start.Pos()), pkg.Fset.Position(end.End())
	lines, ok := assemblyLines[startPosition.Filename]
	if !ok {
		return false
	}
	index, _ := slices.BinarySearch(lines, startPosition.Line)
	if index < len(lines) && lines[index] <= endPosition.Line {
		return false
	}
	return true
}

func AnalyzeModuleAst(
	analysisPath string,
	project []*packages.Package,
	assemblyLines AssemblyLines,
	funcRegistry FuncRegistry,
	policy AnalysisPolicy,
	reporting Reporting,
) error {
	log.Printf("ready to analyze module AST")
	for _, pkg := range project {
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			ctx := GovanishContext{
				Pkg:           pkg,
				AssemblyLines: assemblyLines,
				FuncRegistry:  funcRegistry,
			}
			var currentFunc string
			var analyze func(node ast.Node) bool
			analyze = func(node ast.Node) bool {
				if funcDecl, ok := node.(*ast.FuncDecl); ok {
					currentFunc = funcDecl.Name.Name
				}
				// don't process whole subtree if we should skip the node
				if policy.ShouldSkip(ctx, node) {
					return false
				}
				// process subtree for control-flow pivot nodes but skip analysis of the node itself
				if policy.IsControlFlowPivot(node) {
					return true
				}
				// we can analyze only sequence of statements
				blockStmt, ok := node.(*ast.BlockStmt)
				if !ok {
					return true
				}
				previous := -1
				i := 0
				for i <= len(blockStmt.List) {
					skip1 := i < len(blockStmt.List) && policy.ShouldSkip(ctx, blockStmt.List[i])
					/*
						- we want to also skip patterns like this:
						value, err := F()
						if err != nil {
							...
						}
					*/
					skip2 := i+1 < len(blockStmt.List) && policy.ShouldSkip(ctx, &ast.BlockStmt{List: blockStmt.List[i : i+2]})
					pivot := i == len(blockStmt.List) || policy.IsControlFlowPivot(blockStmt.List[i]) || skip1 || skip2
					// split sequence of statements by pivot positions and analyze regions between them
					if !pivot {
						i += 1
						continue
					}
					if previous+1 < i {
						region := &ast.BlockStmt{List: blockStmt.List[previous+1 : i]}
						start, end := blockStmt.List[previous+1], blockStmt.List[i-1]
						if policy.CheckComplexity(ctx, region) && IsVanished(pkg, assemblyLines, start, end) {
							reporting.ReportVanished(VanishedInfo{
								AnalysisPath: analysisPath,
								Pkg:          pkg,
								FuncName:     currentFunc,
								Start:        start,
								End:          end,
							})
						}
					}
					for s := previous + 1; s < i; s++ {
						ast.Inspect(blockStmt.List[s], analyze)
					}
					if skip1 {
						previous = i
						i += 1
					} else if skip2 {
						previous = i + 1
						i += 2
					} else {
						if i < len(blockStmt.List) {
							ast.Inspect(blockStmt.List[i], analyze)
						}
						previous = i
						i += 1
					}
				}
				return false
			}
			ast.Inspect(file, analyze)
		}
	}
	return nil
}
