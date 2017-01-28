package gogrep

import (
	"fmt"
	"os"
	"regexp"
	"runtime"

	"github.com/simulot/golib/pipeline"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// AppSettings represents application settinsg and run execution
type AppSettings struct {
	regexp       *regexp.Regexp
	string       string
	regEpxSearch bool
	count        bool
	ignoreCase   bool
	lineNumber   bool
	matchOnly    bool
	invertMatch  bool
	numWorker    int
	files        []string
}

// Run the application
func (a *AppSettings) Run() {
	in := make(chan interface{})
	go func() {
		for _, file := range a.files {
			in <- file
		}
		close(in)
	}()

	pipe := pipeline.NewFlow(
		pipeline.GlobOperator(),
		// pipeline.FileDeduplicateOperator(),
		// pipeline.NewParallelFlow(a.numWorker,
		pipeline.FolderToWalkersOperator(),
		pipeline.WalkOperator(),
		a.NewSearcherOperator(),
		// ),
		pipeline.ListerOperator(),
		pipeline.CounterOperator(),
	)

	for i := range pipe.Run(in) {
		fmt.Println(i)
	}
}

// Commandline create command line parameter parser and parse them
func (a *AppSettings) Commandline() error {
	app := kingpin.New("zipgrep", "A zipgrep implementation")
	a.regEpxSearch = true
	app.Flag("count", "Count matching lines").Short('c').BoolVar(&a.count)
	app.Flag("ignore-case", "Ignore case distinction").Short('i').BoolVar(&a.ignoreCase)
	app.Flag("regexp", "PATTERN is a regular expression").Short('e').Action(
		func(c *kingpin.ParseContext) error {
			a.regEpxSearch = true
			return nil
		}).Bool()
	app.Flag("fixed-strings", "PATTERN is a fixed string").Short('F').Action(
		func(c *kingpin.ParseContext) error {
			a.regEpxSearch = false
			return nil
		}).Bool()
	app.Flag("num-worker", "Number of worker to be used").
		Default(fmt.Sprintf("%d", runtime.NumCPU()*2+1)).IntVar(&a.numWorker)
	app.Arg("PATTERN", "PATTERN").StringVar(&a.string)
	app.Arg("file", "files to be searched").StringsVar(&a.files)
	app.Action(func(c *kingpin.ParseContext) error {
		if a.regEpxSearch {
			if re, err := regexp.Compile(a.string); err == nil {
				a.regexp = re
				return nil
			} else {
				return err
			}
		}
		return nil
	})
	_, err := app.Parse(os.Args[1:])
	return err
}
