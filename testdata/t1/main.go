package main

import (
	"fmt"
	"gopkg.in/go-on/builtin.v1"
	"runtime"
)

func main() {
	var b builtin.Bool
	fmt.Printf("%#v (%s)\n", b, runtime.Version())
}
