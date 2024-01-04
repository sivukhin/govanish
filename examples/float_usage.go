//go:build exclude

package main

func FloatUsage(f float64, flag bool) float64 {
	a := -1.
	b := -2.
	c := -3.
	d := -4.
	if flag {
		a = -2.
		b = -3.
		c = -4.
		d = -5.
	}
	return f*a*b*c + f*f*b*c + d
}

func main() {}
