package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	var analysisPath string
	if len(os.Args) == 2 {
		var err error
		analysisPath, err = filepath.Abs(os.Args[1])
		if err != nil {
			panic(fmt.Errorf("unable to expand path '%v' to absolute: %w", os.Args[1], err))
		}
	} else if len(os.Args) == 1 {
		var err error
		analysisPath, err = os.Getwd()
		if err != nil {
			panic(fmt.Errorf("unable to get working directory: %w", err))
		}
	} else {
		panic(fmt.Errorf("usage: govanish | govanish [module path]"))
	}
	log.Printf("module path: %v", analysisPath)
	assemblyLines, err := AnalyzeModuleAssembly(analysisPath)
	if len(assemblyLines) == 0 && err != nil {
		panic(fmt.Errorf("failed to analyze module assembly: %w", err))
	}
	if err != nil {
		log.Printf("module analysis finished with non-critical error: %v", err)
	}
	project, err := LoadPackage(analysisPath)
	if err != nil {
		panic(fmt.Errorf("unable to load project '%v': %w", analysisPath, err))
	}
	funcRegistry := CreateFuncRegistry(project)
	err = AnalyzeModuleAst(project, assemblyLines, funcRegistry, Govanish)
	if err != nil {
		panic(fmt.Errorf("failed to analyze module AST: %w", err))
	}
}
