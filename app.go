package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/ttacon/chalk"
	"golang.org/x/sync/errgroup"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var isWindowsOS = runtime.GOOS == "windows"

// App represents application settings and run execution
type App struct {
	regexp       *regexp.Regexp
	string       string
	regEpxSearch bool
	count        bool
	ignoreCase   bool
	numWorker    int
	mask         string
	useColors    bool
	colorSet     colorSet
	files        []string

	bytesRead   int64
	filesParsed int64
	hitCount    int64

	limiter *Limiter
}

type colorSet struct {
	archive, file, line, unmatched, matched func(string) string
}

type Limiter struct {
	limiter chan bool
}

func NewLimiter(number int) *Limiter {
	l := Limiter{
		limiter: make(chan bool, number),
	}

	return &l
}

func (l *Limiter) Start() {
	l.limiter <- true
}

func (l *Limiter) Done() {
	<-l.limiter
}

// Run the application
func (a *App) Run() error {
	a.limiter = NewLimiter(a.numWorker)
	group := errgroup.Group{}

	// one go routine per OS file
	for _, arg := range a.files {
		files, err := filepath.Glob(arg)
		if err != nil {
			return err
		}
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				return err
			}
			if info.IsDir() {
				fsys := os.DirFS(file)
				fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
					if d.IsDir() {
						return nil
					}
					a.limiter.Start()
					group.Go(func() error {
						defer func() {
							a.limiter.Done()
						}()
						f, err := fsys.Open(path)
						if err != nil {
							return err
						}
						defer f.Close()
						err = a.ProcessFile(f, "")
						return err
					})
					return nil
				})
				continue
			}

			a.limiter.Start()
			file := file
			group.Go(func() error {
				defer func() {
					a.limiter.Done()
				}()
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()
				err = a.ProcessFile(f, "")
				if err != nil {
					err = fmt.Errorf("can't process os file '%s', %w", file, err)
				}
				return err
			})
		}
	}
	return group.Wait()
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
	app := kingpin.New("zipgrep", "A zipgrep implementation")
	a.regEpxSearch = true
	app.Flag("count", "Count matching lines").Short('c').BoolVar(&a.count)
	app.Flag("ignore-case", "Ignore case distinction").Short('i').BoolVar(&a.ignoreCase)
	app.Flag("regexp", "PATTERN is a regular expression").Short('e').Action(
		func(c *kingpin.ParseContext) error {
			a.regEpxSearch = true
			return nil
		}).Bool()
	app.Flag("string", "PATTERN is a string").Short('s').Action(
		func(c *kingpin.ParseContext) error {
			a.regEpxSearch = false
			return nil
		}).Bool()
	app.Flag("num-worker", "Number of worker to be used").
		Default(fmt.Sprintf("%d", runtime.NumCPU())).IntVar(&a.numWorker)
	app.Flag("mask", "search only in files following the mask inside archive file").Short('m').Default("*.*").
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
	app.Arg("file", "files, folder or archive to be searched").StringsVar(&a.files)
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
				archive:   chalk.Cyan.Color,
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
