package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeFunc(t *testing.T) {
	t.Run("simple func", func(t *testing.T) {
		dir, dispose, err := MustGenMod(`package main
func Run(n int) string {
if n == 0 {
	return "0"
} else if n == 1 {
	return "1"
} else {
	return "2"
}
}
`)
		defer dispose()
		project, err := LoadPackage(dir)
		require.Nil(t, err)
		require.Equal(t, FuncProps{DeterministicReturn: false}, analyzeFunc(project[0], MustExtractFunc(project)))
	})
	t.Run("ignore wrapped calls", func(t *testing.T) {
		dir, dispose, err := MustGenMod(`package main
func Run(n int) string {
	f := func() string {
		return "0"
	}
	return f()
	}
	`)
		defer dispose()
		project, err := LoadPackage(dir)
		require.Nil(t, err)
		require.Equal(t, FuncProps{DeterministicReturn: false}, analyzeFunc(project[0], MustExtractFunc(project)))
	})
	t.Run("deterministic return", func(t *testing.T) {
		dir, dispose, err := MustGenMod(`package main
func Run(n int) string {
	return "1"
}
func main() {}
	`)
		defer dispose()
		project, err := LoadPackage(dir)
		require.Nil(t, err)
		require.Equal(t, FuncProps{DeterministicReturn: true}, analyzeFunc(project[0], MustExtractFunc(project)))
	})
	t.Run("deterministic nil return", func(t *testing.T) {
		dir, dispose, err := MustGenMod(`package main
func Write(b *strings.Builder, s string) error {
	_, _ = b.WriteString(s)
	return nil
}
func main() {}
`)
		defer dispose()
		project, err := LoadPackage(dir)
		require.Nil(t, err)
		require.Equal(t, FuncProps{DeterministicReturn: true}, analyzeFunc(project[0], MustExtractFunc(project)))
	})
}
