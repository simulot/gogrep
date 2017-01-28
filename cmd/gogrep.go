package main

import (
	"fmt"
	"os"

	"github.com/pkg/profile"
	"github.com/simulot/gogrep/gogrep"
	// Register common walkers
	_ "github.com/simulot/golib/file/walker/zipwalker"
)

func main() {
	defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()
	app := &gogrep.AppSettings{}
	err := app.Commandline()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	app.Run()
}
