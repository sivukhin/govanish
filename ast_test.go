package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericFunc(t *testing.T) {
	t.Run("generic T any", func(t *testing.T) {
		_, f := MustGenFunc(`func G[T any]() { }`)
		require.True(t, IsGenericFunc(f))
	})
	t.Run("generic T ~Scanner", func(t *testing.T) {
		_, f := MustGenFunc(`func G[T ~Scanner]() { }`)
		require.True(t, IsGenericFunc(f))
	})
	t.Run("generic T ****any", func(t *testing.T) {
		_, f := MustGenFunc(`func G[T ****any]() { }`)
		require.True(t, IsGenericFunc(f))
	})
	t.Run("generic T, Q, K ****any", func(t *testing.T) {
		_, f := MustGenFunc(`func G[T, Q, K ****any]() { }`)
		require.True(t, IsGenericFunc(f))
	})
	t.Run("receiver T", func(t *testing.T) {
		_, f := MustGenFunc(`func (s Q[T]) G() { }`)
		require.True(t, IsGenericFunc(f))
	})
	t.Run("receiver T, Q, K", func(t *testing.T) {
		_, f := MustGenFunc(`func (s Q[T, Q, K]) G() { }`)
		require.True(t, IsGenericFunc(f))
	})
	t.Run("non-generic", func(t *testing.T) {
		_, f := MustGenFunc(`func (s Q) G() { }`)
		require.False(t, IsGenericFunc(f))
	})
}

func TestEqualExprs(t *testing.T) {
	t.Run("idents", func(t *testing.T) {
		_, e0 := MustGenExpr(`a.Field.Value`)
		_, e1 := MustGenExpr(`a.Field.Value`)
		_, e2 := MustGenExpr(`b.Field.Value`)
		_, e3 := MustGenExpr(`a.Prop.Value`)
		_, e4 := MustGenExpr(`a.Field.Key`)
		require.True(t, EqualExprs(e0, e1))
		require.True(t, EqualExprs(e1, e1))
		require.False(t, EqualExprs(e1, e2))
		require.False(t, EqualExprs(e1, e3))
		require.False(t, EqualExprs(e1, e4))
	})
	t.Run("array access", func(t *testing.T) {
		_, e0 := MustGenExpr(`a.Field[0].Value`)
		_, e1 := MustGenExpr(`a.Field[0].Value`)
		_, e2 := MustGenExpr(`a.Field[1].Value`)
		_, e3 := MustGenExpr(`a.Field[b].Value`)
		require.True(t, EqualExprs(e0, e1))
		require.False(t, EqualExprs(e1, e2))
		require.False(t, EqualExprs(e1, e3))
	})
}

func TestDeconstructSelector(t *testing.T) {
	t.Run("simple selector", func(t *testing.T) {
		_, e := MustGenExpr(`a.Field.Value`)
		selector, ok := DeconstructSelector(e)
		require.True(t, ok)
		require.Equal(t, "a.Field.Value", selector)
	})
}
