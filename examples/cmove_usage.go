//go:build exclude

package main

func CMove(n, m int, flag bool) int {
	if flag {
		// this line is deleted from assembly but this is due to the rewrite of the function with CMOVQNE instruction
		n = m + m
	}
	return n
}

func main() {}
