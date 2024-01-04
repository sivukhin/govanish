//go:build exclude

package main

import "fmt"

type E struct{ Desc string }

func (e *E) Error() string { return e.Desc }
func Api(n int) *E {
	if n == 0 {
		return nil
	}
	return &E{Desc: "error"}
}

func BoxingErr(n int) {
	var err error = Api(n)
	if err != nil {
		panic("this is impossible")
	}
	// this line vanished because Go compiler proves that err always not nil because of boxing
	panic(fmt.Sprintf("this is possible: %v", err))
}

func main() {}
