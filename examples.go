package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
)

type Alias int

type Wr interface {
	Write([]byte) (int64, error)
}

func WriteInflate(w Wr, p []byte) (int, error) {
	n, err := w.Write(p)
	nn := int(n)
	if int64(nn) != n {
		return 0, fmt.Errorf("too much data inflated: %d", n)
	}
	return nn, err
}

func check3(a, b []string) bool {
	changed := true
	func() {
		if len(a) != len(b) {
			return
		}
		for i, value := range a {
			if value != b[i] {
				return
			}
		}
		changed = false
	}()
	return changed
}

func check(n int) bool {
	a := Alias(n)
	if a > 1 {
		return true
	}
	return false
}

func check2(n int) Alias {
	a := Alias(n)
	if a > 1 {
		a = a + 1
	} else {
		a = Alias(1)
	}
	return a
}

type S struct {
	A int
	B string
	C int
}

func run(n int) {
	s := S{A: n}
	if s.A == 1 {
		panic("s.A == 1")
	}
}

func d(ctx context.Context) bool {
	c := make(chan int)
	go func() {
		c <- 1
	}()
	select {
	case <-c:
		return true
	case <-ctx.Done():
		return false
	}
}

func c(n int) int {
	ctx := context.Background()
	if d(ctx) {
		return n
	}
	return 0
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

type E struct{ Desc string }

func (e *E) Error() string { return e.Desc }
func api(n int) *E {
	if n == 0 {
		return nil
	}
	return &E{Desc: "error"}
}

func use(n int) {
	var err error = api(n)
	if err != nil {
		panic("this is impossible")
	}
	panic(fmt.Sprintf("this is possible: %v", err))
}
