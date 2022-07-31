package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
	regexp     *regexp.Regexp
	count      bool
	ignoreCase bool
	numWorker  int
	mask       string
	useColors  bool
	colorSet   colorSet
	files      []string

	bytesRead   int64
	filesParsed int64
	hitCount    int64
	group       *errgroup.Group
}

type colorSet struct {
	archive, file, line, unmatched, matched func(string) string
}

// Run the application
func (a *App) Run(ctx context.Context) error {
	a.group, ctx = errgroup.WithContext(ctx)
	if a.numWorker > 0 {
		a.group.SetLimit(a.numWorker)
	} else {
		a.group.SetLimit(1)
	}
	lfs, err := listfs.Open(a.files)
	if err != nil {
		return err
	}
	err = a.ProcessArchive(ctx, lfs, "")
	if err != nil {
		return err
	}
	return a.group.Wait()
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

	r, err := regexp.Compile(args[0])
	if err != nil {
		return a.Usage(err)
	}

	a.regexp = r
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
	fmt.Println("\tPATTERN is a regular expression")
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
