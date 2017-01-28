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
	file         walker.WalkItem
	occurrrences []Occurrence
}

type Occurrence struct {
	line  int
	index [][]int
	text  string
}

func (f *FoundItem) String() string {
	s := ""
	for _, o := range f.occurrrences {
		s += fmt.Sprintf("%s:%d:%s\n", f.file.FullName(), o.line, o.text)
	}
	return s
}

func (a *AppSettings) NewSearcherOperator() pipeline.Operator {
	const errorContext = "NewSearcher Operator"
	strPattern := []byte(a.string)
	return func(in chan interface{}, out chan interface{}) {
		for i := range in {
			item, ok := i.(walker.WalkItem)
			found := &FoundItem{
				file: item,
			}

			if !ok {
				panic("Expecting *WalkItem in " + errorContext)
			}
			itemReader, err := item.Reader()
			if err != nil {
				fmt.Fprint(os.Stderr, errors.Wrap(err, "Can read item"))
				item.Close()
				continue
			}
			r := bufio.NewReaderSize(itemReader, 32*1024)
			lineNumber := 0
			text, err := r.ReadBytes('\n')
			stext := []byte{}
			for err == nil || err == io.EOF {
				debug := string(text)
				_ = debug
				lineNumber++
				if !a.regEpxSearch {
					if a.ignoreCase {
						stext = bytes.ToUpper(text)
					} else {
						stext = text
					}
					if pos := bytes.Index(stext, strPattern); pos >= 0 {
						found.occurrrences = append(found.occurrrences, Occurrence{
							line:  lineNumber,
							text:  stripLn(text),
							index: [][]int{{pos, pos + len(a.string)}},
						})
					}
				} else {
					if indexes := a.regexp.FindAllSubmatchIndex(text, -1); len(indexes) > 0 {
						found.occurrrences = append(found.occurrrences, Occurrence{
							line:  lineNumber,
							index: indexes,
							text:  stripLn(text),
						})
					}
				}

				text, err = r.ReadBytes('\n')
				if err == io.EOF {
					if len(found.occurrrences) == 0 {
						item.Close()
					} else {
						out <- item
					}
					break
				}

				if err != nil && err != io.EOF {
					fmt.Fprintln(os.Stderr, errors.Wrapf(err,
						"%s: Can't scan file in search (%s, line %d):\n%s",
						errorContext, item.FullName(), lineNumber, string(text)))
					item.Close()
					continue
				}
			}
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
