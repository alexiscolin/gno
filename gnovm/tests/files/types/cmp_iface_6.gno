package main

import (
	"strconv"
)

type E interface {
	Error() string
}

type Error1 int64

func (e Error1) Error() string {
	return "error: " + strconv.Itoa(int(e))
}

type Error2 int64

func (e Error2) Error() string {
	return "error: " + strconv.Itoa(int(e))
}

// both not const, and both interface
func main() {
	var e1 E = Error1(0)
	var e2 E = Error2(0)
	println(e1 == e2)
}

// Output:
// false
