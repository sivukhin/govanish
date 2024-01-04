package main

import (
	_ "embed"
	"go/ast"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestAnalyzeModuleAssembly(t *testing.T) {
	t.Run("simple assembly", func(t *testing.T) {
		path, dispose, err := MustGenMod(`
package main
import "fmt"
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

func main() {}`)
		require.Nil(t, err)
		defer dispose()

		assemblyLines, err := AnalyzeModuleAssembly(path)
		require.Nil(t, err)
		t.Log(assemblyLines)
		require.Len(t, assemblyLines, 1)
		var lines []int
		for _, value := range assemblyLines {
			lines = value
		}
		require.Equal(t, []int{
			6, 7, 8, 9, 11, /* api */
			14, 15, 17, /* use */
			22, /* main */
		}, lines)
	})
	t.Run("instantiated generics", func(t *testing.T) {
		path, dispose, err := MustGenMod(`
package main
import "fmt"

type E[T any] struct{ Desc T }

func (r E[T]) Run() { 
	fmt.Printf("%v", r.Desc)
}

func main() {
	E[string]{Desc: "Hello"}.Run()
	E[int]{Desc: 10}.Run()
}`)
		require.Nil(t, err)
		defer dispose()

		assemblyLines, err := AnalyzeModuleAssembly(path)
		require.Nil(t, err)
		require.Len(t, assemblyLines, 1)
		var lines []int
		for _, value := range assemblyLines {
			lines = value
		}
		require.Equal(t, []int{7, 8, 9, 11, 12, 13, 14}, lines)
	})
	t.Run("not instantiated generics", func(t *testing.T) {
		path, dispose, err := MustGenMod(`
package main
import "fmt"

type E[T any] struct{ Desc T }

func (r E[T]) Run() { 
	fmt.Printf("%v", r.Desc)
}

func main() {
	fmt.Printf("%v", "Hello")
	fmt.Printf("%v", 10)
}`)
		require.Nil(t, err)
		defer dispose()

		assemblyLines, err := AnalyzeModuleAssembly(path)
		require.Nil(t, err)
		require.Len(t, assemblyLines, 1)
		var lines []int
		for _, value := range assemblyLines {
			lines = value
		}
		require.Equal(t, []int{11, 12, 13, 14}, lines)
	})
}

type testPolicy struct {
	Vanished []simpleVanishedInfo
}

func (t *testPolicy) ShouldSkip(pkg *packages.Package, node ast.Node) bool {
	return Govanish.ShouldSkip(pkg, node)
}
func (t *testPolicy) IsControlFlowPivot(node ast.Node) bool { return Govanish.IsControlFlowPivot(node) }
func (t *testPolicy) CheckComplexity(node ast.Node) bool    { return Govanish.CheckComplexity(node) }
func (t *testPolicy) ReportVanished(info VanishedInfo) {
	t.Vanished = append(t.Vanished, simpleVanishedInfo{
		Func:      info.FuncName,
		StartLine: info.Pkg.Fset.Position(info.Start.Pos()).Line,
		EndLine:   info.Pkg.Fset.Position(info.End.Pos()).Line,
	})
}

type simpleVanishedInfo struct {
	Func      string
	StartLine int
	EndLine   int
}

func analyze(t *testing.T, src string) []simpleVanishedInfo {
	path, dispose, err := MustGenMod(src)
	require.Nil(t, err)
	defer dispose()

	assemblyLines, err := AnalyzeModuleAssembly(path)
	require.Nil(t, err)

	policy := &testPolicy{}
	require.Nil(t, AnalyzeModule(path, assemblyLines, policy))
	return policy.Vanished
}

var excludeComment = "//go:build exclude\n\n"

func loadExample(t *testing.T) string {
	tokens := strings.Split(t.Name(), "/")
	data, err := os.ReadFile(path.Join("examples", tokens[len(tokens)-1]))
	require.Nil(t, err)
	return strings.TrimPrefix(string(data), excludeComment)
}

func TestAnalysis(t *testing.T) {
	t.Run("err_not_nil_tricky_bug.go", func(t *testing.T) {
		vanished := analyze(t, loadExample(t))
		require.Equal(t, []simpleVanishedInfo{{Func: "BoxingErr", StartLine: 21, EndLine: 21}}, vanished)
	})
	t.Run("platform_dependent_code_trick.go", func(t *testing.T) {
		vanished := analyze(t, loadExample(t))
		require.Equal(t, []simpleVanishedInfo{{Func: "WritePlatformDependent", StartLine: 12, EndLine: 12}}, vanished)
	})
	t.Run("forgotten_errcheck_bug.go", func(t *testing.T) {
		vanished := analyze(t, loadExample(t))
		require.Equal(t, []simpleVanishedInfo{{Func: "NoErrCheck", StartLine: 11, EndLine: 11}}, vanished)
	})
	t.Run("platform_dependent_code.go", func(t *testing.T) {
		vanished := analyze(t, loadExample(t))
		require.Empty(t, vanished)
	})
	t.Run("var_check_elimination.go", func(t *testing.T) {
		vanished := analyze(t, loadExample(t))
		require.Equal(t, []simpleVanishedInfo{{Func: "LongCompute", StartLine: 19, EndLine: 19}}, vanished)
	})
}
