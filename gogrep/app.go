package gogrep

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"

	"github.com/pkg/errors"
	"github.com/simulot/golib/pipeline"
	"github.com/ttacon/chalk"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var isWindowsOS = runtime.GOOS == "windows"

// AppSettings represents application settings and run execution
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
	mask         string
	useColors    bool
	colorSet     colorSet
	files        []string
}

type colorSet struct {
	archive, file, line, unmatched, matched func(string) string
}

type Worker struct {
	limiter chan interface{}
}

func NewWorker(number int) (*Worker, func()) {
	w := make(chan bool, number)
	for i := 0 ; i<number;i++{
		w <- interface{}{}
	}
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
		pipeline.FolderToWalkersOperator(),
		pipeline.NewParallelFlow(a.numWorker,
			pipeline.WalkOperator(),
			pipeline.FileMaskOperator(a.mask),
			a.NewSearcherOperator(),
		),
		a.OutputOperator(),
		// pipeline.ListerOperator(),
		// pipeline.CounterOperator(),
	)

	for i := range pipe.Run(in) {
		fmt.Println(i)
	}
}

func (a *AppSettings) OutputOperator() pipeline.Operator {
	return func(in, ou chan interface{}) {
		for i := range in {
			if item, ok := i.(*FoundItem); ok {
				a.OutputFoundItem(item)
			} else {
				panic("Expecting a *FoundItem in search.OutputOperator")
			}
		}

	}
}

func (a *AppSettings) OutputFoundItem(f *FoundItem) {
	for _, o := range f.occurrences {
		if a.useColors {
			fmt.Fprint(os.Stdout, a.colorSet.file(f.file.FullName()))
			fmt.Fprint(os.Stdout, ":")
			fmt.Fprint(os.Stdout, a.colorSet.line(strconv.Itoa(o.line)))
			fmt.Fprint(os.Stdout, ":")
			i := 0
			t := o.text
			for _, m := range o.index {
				if i < m[0] {
					fmt.Fprint(os.Stdout, a.colorSet.unmatched(t[i:m[0]]))
				}
				fmt.Fprint(os.Stdout, a.colorSet.matched(t[m[0]:m[1]]))
				i = m[1]
			}
			if i < len(t) {
				fmt.Fprint(os.Stdout, a.colorSet.unmatched(t[i:]))
			}
			fmt.Fprint(os.Stdout, "\n")
		} else {
			fmt.Fprintf(os.Stdout, "%s:%d:%s\n", f.file.FullName(), o.line, o.text)
		}
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
		Default(fmt.Sprintf("%d", runtime.NumCPU())).IntVar(&a.numWorker)
	app.Flag("mask", "search only in files following the mask").Short('m').Default("*.*").
		Action(func(c *kingpin.ParseContext) error {
			if len(a.mask) > 0 && a.mask != "*.*" {
				if _, err := filepath.Match(a.mask, "test.file"); err != nil {
					return errors.Wrapf(err, "Invalid --mask option")
				}
			}
			return nil
		}).StringVar(&a.mask)
	colorFlag := "true"
	if isWindowsOS {
		colorFlag = "false"
	}
	app.Flag("color", "Use colors").Default(colorFlag).BoolVar(&a.useColors)
	app.Arg("PATTERN", "PATTERN").StringVar(&a.string)
	app.Arg("file", "files to be searched").StringsVar(&a.files)
	app.Action(func(c *kingpin.ParseContext) error {
		if a.regEpxSearch {
			if re, err := regexp.Compile(a.string); err == nil {
				a.regexp = re
			} else {
				return err
			}
		}
		if a.useColors {
			a.colorSet = colorSet{
				archive:   chalk.Magenta.Color,
				file:      chalk.Magenta.Color,
				line:      chalk.Green.Color,
				unmatched: chalk.White.Color,
				matched:   chalk.Yellow.Color,
			}
		}
		return nil
	})
	_, err := app.Parse(os.Args[1:])
	return err
}
