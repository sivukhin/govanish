//go:build exclude

package main

import (
	"fmt"
)

func LongCompute(b []int) {
	if b[0] != 10000 {
		return
	}
	sum := int64(0)
	for x := 0; x < 10000; x++ {
		for y := 0; y < 10000; y++ {

		}
	}
	if b[0] == 0 {
		// this line removed because b[0] == 10000 due to the check at line 10
		panic("b[0] == 0")
	}
	fmt.Printf("b[0] = %v, sum = %v", b[0], sum)
}

func main() {}
