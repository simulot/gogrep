package main

import (
	"fmt"
	"os"

	// Register common walkers
	"github.com/simulot/gogrep/gogrep"
	_ "github.com/simulot/golib/file/walker/zip"
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
