package main

import (
	"fmt"
	"os"

	"github.com/simulot/gogrep/cmd/gogrep"
	// Register common walkers
	_ "github.com/simulot/golib/file/walker/zipwalker"
)

func main() {
	app := &gogrep.AppSettings{}
	err := app.Commandline()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	app.Run()
}
