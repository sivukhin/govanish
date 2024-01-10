//go:build exclude

package main

import (
	"fmt"
	"time"
)

func ConstExprUsage(flag bool) {
	delay := time.Hour * 10 * 7
	var pos int
	if flag {
		time.Sleep(delay)
		pos = 1
	} else {
		pos = 2
	}
	fmt.Printf("pos = %v", pos)
}

func main() {}
