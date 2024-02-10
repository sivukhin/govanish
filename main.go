package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	modulePath := flag.String("path", "", "path to the module root (with go.mod file)")
	reportFormat := flag.String("format", "log", "reporting type (github | log)")
	flag.Parse()

	var reporting Reporting
	if *reportFormat == "github" {
		reporting = GitHubReporting{}
	} else if *reportFormat == "log" {
		reporting = LogReporting{}
	} else {
		fmt.Printf("invalid -format value: %v\n", *reportFormat)
		flag.Usage()
		os.Exit(1)
	}

	var analysisPath string
	var err error
	if *modulePath == "" {
		analysisPath, err = os.Getwd()
		if err != nil {
			fmt.Printf("unable to get working directory: %v\n", err)
			flag.Usage()
			os.Exit(1)
		}
	} else {
		analysisPath, err = filepath.Abs(*modulePath)
		if err != nil {
			fmt.Printf("unable to expand path '%v' to absolute: %v\n", *modulePath, err)
			flag.Usage()
			os.Exit(1)
		}
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

	err = AnalyzeModuleAst(analysisPath, project, assemblyLines, funcRegistry, Govanish, reporting)
	if err != nil {
		panic(fmt.Errorf("failed to analyze module AST: %w", err))
	}
}
