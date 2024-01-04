//go:build exclude

package main

import (
	"unsafe"
)

func CastUsage(i uint32) {
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		panic("1")
	} else {
		panic("2")
	}
}

func main() {}
