//go:build exclude

package main

import (
	"fmt"
	"io"
	"os"
)

func InterfaceCast(name string, size int) ([]byte, error) {
	buffer := make([]byte, size)
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	reader := (io.Reader)(f)
	if size < 1024 {
		return nil, fmt.Errorf("buffer is too small")
	}
	n, err := reader.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

func main() {}
