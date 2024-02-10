package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Reporting interface{ ReportVanished(info VanishedInfo) }

type LogReporting struct{}

func (_ LogReporting) ReportVanished(info VanishedInfo) {
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

type GitHubReporting struct{}

func (_ GitHubReporting) ReportVanished(info VanishedInfo) {
	relativePath, _ := filepath.Rel(info.AnalysisPath, info.Filename())
	fmt.Printf("::warning file=%v,line=%v,endLine=%v::%v\n", relativePath, info.StartLine(), info.EndLine(), "seems like code vanished from compiled binary")
}
