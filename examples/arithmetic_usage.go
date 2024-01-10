//go:build exclude

package main

func ArithmeticUsage(n []int, s int) int {
	sum := 0
	if offset := s % 64; offset > 0 {
		// here golang reuse computation from previous line so it seems like it vanished from the executable
		start := (s / 64) * 64
		for i := offset; i < start; i++ {
			sum += n[i]
		}
	} else {
		for _, value := range n {
			sum += value
		}
	}
	return sum
}

func main() {}
