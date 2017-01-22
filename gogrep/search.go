package gogrep

import (
	"bufio"
	"bytes"
	"io"

	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/simulot/golib/file/walker"
	"github.com/simulot/golib/pipeline"
)

type FoundItem struct {
	file  walker.ItemOpenCloser
	line  int
	index [][]int
	text  string
}

func (f *FoundItem) String() string {
	return fmt.Sprintf("%s:%d:%s", f.file.FullName(), f.line, f.text)
}

func (a *AppSettings) NewSearcherOperator() pipeline.Operator {
	const errorContext = "NewSearcher Operator"
	strPattern := []byte(a.string)
	return func(in chan interface{}, out chan interface{}) {
		for i := range in {
			func() {
				file, ok := i.(walker.ItemOpenCloser)
				if !ok {
					panic("Expecting *MatchedItem in " + errorContext)
				}
				f, err := file.Open()
				if err != nil {
					fmt.Fprintln(os.Stderr, errors.Wrap(err, errorContext))
					file.Done()
					return
				}

				if f == nil {
					panic("f is nil in " + errorContext)
				}

				defer f.Close()
				defer file.Done()

				r := bufio.NewReaderSize(f, 32*1024)
				lineNumber := 0
				text, err := r.ReadBytes('\n')
				stext := []byte{}
				for err == nil || err == io.EOF {
					lineNumber++
					if !a.regEpxSearch {
						if a.ignoreCase {
							stext = bytes.ToUpper(text)
						} else {
							stext = text
						}
						if pos := bytes.Index(stext, strPattern); pos >= 0 {
							out <- &FoundItem{
								file:  file,
								line:  lineNumber,
								text:  stripLn(text),
								index: [][]int{{pos, pos + len(a.string)}},
							}
						}
					} else {
						if indexes := a.regexp.FindAllSubmatchIndex(text, -1); len(indexes) > 0 {
							out <- &FoundItem{
								file:  file,
								line:  lineNumber,
								index: indexes,
								text:  stripLn(text),
							}
						}
					}

					if err == io.EOF {
						return
					}
					text, err = r.ReadBytes('\n')

					if err != nil && err != io.EOF {
						fmt.Fprintln(os.Stderr, errors.Wrapf(err,
							"%s: Can't scan file in search (%s, line %d):\n%s",
							errorContext, file.FullName(), lineNumber, string(text)))
						return
					}
				}
			}()
		}
	}
}

func stripLn(s []byte) string {
	p := len(s) - 1
	l := p
	for p >= 0 && l-p <= 2 {
		switch s[p] {
		case '\n', '\r':
			p--
		default:
			return string(s[:p+1])
		}
	}
	return string(s)
}
