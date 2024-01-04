//go:build exclude

package main

func NoErrCheck(w interface{ Write(n int) error }) {
	err := w.Write(1)
	if err != nil {
		panic(err)
	}
	_ = w.Write(2)
	if err != nil {
		// this line removed by compiler because err were already checked on line 6 and didn't changed since that
		panic(err)
	}
}

func main() {}
