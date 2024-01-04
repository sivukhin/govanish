//go:build exclude

package main

const A = 1
const B = 2

func ConstDecl(n int, flag bool) int {
	// this line is removed but this is ok - it's constant
	const C = A + B*A
	if flag {
		panic("no-no-no")
	}
	return n
}

func main() {}
