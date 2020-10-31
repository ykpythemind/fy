package main

import (
	"fmt"
	"os"

	"github.com/ykpythemind/fy"
)

func main() {
	app, err := fy.New()
	if err != nil {
		fmt.Fprint(os.Stdout, err)
		os.Exit(1)
	}

	err = app.Run()
	if err != nil {
		fmt.Fprint(os.Stdout, err)
		os.Exit(1)
	}
}
