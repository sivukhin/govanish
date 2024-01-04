//go:build exclude

package main

import (
	"fmt"
)

func WritePlatformDependent(w interface{ Write([]byte) (int64, error) }, p []byte) (int, error) {
	n, err := w.Write(p)
	nn := int(n)
	if int64(nn) != n {
		// this line is removed on 64bit architectures
		return 0, fmt.Errorf("too much data: %d", n)
	}
	return nn, err
}

func main() {}
