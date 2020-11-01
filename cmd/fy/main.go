package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ykpythemind/fy"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug")

	flag.Parse()

	app, err := fy.New(os.Args, debug)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	err = app.Run()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
