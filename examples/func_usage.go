//go:build exclude

package main

import (
	"io"
)

func FuncUsage(r io.Reader) []byte {
	return func() []byte {
		if r == nil {
			return nil
		}
		buffer := make([]byte, 1024)
		_, _ = r.Read(buffer)
		return buffer
	}()
}

func main() {}
