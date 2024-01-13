//go:build exclude

package main

import (
	"strings"
)

func Write(b *strings.Builder, s string) error {
	_, _ = b.WriteString(s)
	return nil
}

func ConstReturnLib() string {
	var b strings.Builder
	err := Write(&b, "hello")
	if err != nil {
		// this line is vanished because WriteString always return nil err
		panic(err)
	}
	return b.String()
}

func main() {}
