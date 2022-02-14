package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	// defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()
	app := &App{}
	err := app.Commandline()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	t0 := time.Now()
	err = app.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	d := time.Since(t0).Round(time.Millisecond)
	fmt.Println("Hits", app.hitCount)
	fmt.Println("Total time", d)
	fmt.Println("File parsed", app.filesParsed)
	fmt.Println("Total", ByteCounter(app.bytesRead))
	fmt.Println("Speed", BytePerSecond{app.bytesRead, d})
}
