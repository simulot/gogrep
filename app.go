package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"runtime"
	"sync/atomic"

	"github.com/simulot/gogrep/listfs"
	"github.com/ttacon/chalk"
	"golang.org/x/sync/errgroup"
)

var isWindowsOS = runtime.GOOS == "windows"

// App represents application settings and run execution
type App struct {
	regexp          *regexp.Regexp
	string          string
	stringExpSearch bool
	count           bool
	ignoreCase      bool
	numWorker       int
	mask            string
	useColors       bool
	colorSet        colorSet
	files           []string

	bytesRead   int64
	filesParsed int64
	hitCount    int64
	group       errgroup.Group
	limiter     *Limiter
}

type colorSet struct {
	archive, file, line, unmatched, matched func(string) string
}

// Run the application
func (a *App) Run() error {
	lfs, err := listfs.Open(a.files)
	if err != nil {
		return err
	}
	return a.ProcessArchive(lfs, "")
}

func (a *App) IsMatch(f string) bool {
	m, _ := filepath.Match(a.mask, filepath.Base(f))
	return m
}

type Hit struct {
	Archive    string
	File       string
	LineNumber int
	Line       string
	Loc        []int
}

func (a *App) OutputHit(h Hit) {
	atomic.AddInt64(&a.hitCount, 1)
	if a.useColors {
		if len(h.Archive) > 0 {
			fmt.Print(a.colorSet.archive(h.Archive) + ":")
		}
		fmt.Print(a.colorSet.file(h.File), a.colorSet.line(fmt.Sprintf("(%d)", h.LineNumber))+":")
		i := 0
		for m := 0; m < len(h.Loc); m += 2 {
			fmt.Print(a.colorSet.unmatched(h.Line[i:h.Loc[m]]))
			fmt.Print(a.colorSet.matched(h.Line[h.Loc[m]:h.Loc[m+1]]))
			i += h.Loc[m+1]
		}
		fmt.Println(a.colorSet.unmatched(h.Line[i:]))
		return
	}
	if len(h.Archive) > 0 {
		fmt.Print(h.Archive + ":")
	}
	fmt.Println(h.File + ":" + h.Line)
}

// Commandline create command line parameter parser and parse them
func (a *App) Commandline() error {
	flag.BoolVar(&a.count, "count", false, "Count matching lines")
	flag.BoolVar(&a.ignoreCase, "ignore-case", false, "Ignore case distinction")
	flag.BoolVar(&a.stringExpSearch, "s", true, "PATTERN is a simple string")
	flag.IntVar(&a.numWorker, "num-worker", runtime.NumCPU(), "Number of worker to be used")
	flag.StringVar(&a.mask, "mask", "*.*", "search only in files following the mask inside archive file")
	flag.BoolVar(&a.useColors, "color", !isWindowsOS, "Use colored outputs")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return a.Usage(errors.New("missing PATTERN"))
	}
	if len(args) < 2 {
		return a.Usage(errors.New("missing file or dir"))
	}

	a.files = args[1:]

	if a.stringExpSearch {
		a.string = args[0]
	} else {
		r, err := regexp.Compile(args[0])
		if err != nil {
			return a.Usage(err)
		}
		a.regexp = r
	}
	if a.useColors {
		a.colorSet = colorSet{
			archive:   chalk.Cyan.Color,
			file:      chalk.Magenta.Color,
			line:      chalk.Green.Color,
			unmatched: chalk.White.Color,
			matched:   chalk.Yellow.Color,
		}
	}
	return nil
}

func (a App) Usage(e error) error {
	fmt.Println("gogrep {option, ...} PATTERN FileOrDir, ...")
	fmt.Println("\tPATTERN could be a regular expression or a simple string")
	flag.Usage()
	if e != nil {
		fmt.Println(e.Error())
	}
	return e
}

func (a *App) CountReader(r io.Reader) io.Reader {
	return &CountReader{a: a, Reader: r}
}

type CountReader struct {
	a *App
	io.Reader
	count int
}

func (cr *CountReader) Read(b []byte) (int, error) {
	n, err := cr.Reader.Read(b)
	cr.count += n
	if err != nil || cr.count >= 10*1024 {
		atomic.AddInt64(&cr.a.bytesRead, int64(cr.count))
		cr.count = 0
	}
	return n, err
}
