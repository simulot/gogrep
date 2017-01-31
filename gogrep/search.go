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
	file        walker.WalkItem
	occurrences []Occurrence
}

type Occurrence struct {
	line  int
	index [][]int
	text  string
}

func (f *FoundItem) String() string {
	s := ""
	crlf := "\n"
	for i, o := range f.occurrences {
		if i == len(f.occurrences)-1 {
			crlf = ""
		}
		s += fmt.Sprintf("%s:%d:%s%s", f.file.FullName(), o.line, o.text, crlf)
	}
	return s
}

// NewSearcherOperator takes a walker.Item, gets a reader on it
// and search in its content
// IN: walker.Item
// OUT: FoundItem
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
			sText := []byte{}
			for err == nil || err == io.EOF {
				debug := string(text)
				_ = debug
				lineNumber++
				if !a.regEpxSearch {
					if a.ignoreCase {
						sText = bytes.ToUpper(text)
					} else {
						sText = text
					}
					if pos := bytes.Index(sText, strPattern); pos >= 0 {
						found.occurrences = append(found.occurrences, Occurrence{
							line:  lineNumber,
							text:  stripLn(text),
							index: [][]int{{pos, pos + len(a.string)}},
						})
					}
				} else {
					if indexes := a.regexp.FindAllIndex(text, -1); len(indexes) > 0 {
						found.occurrences = append(found.occurrences, Occurrence{
							line:  lineNumber,
							index: indexes,
							text:  stripLn(text),
						})
					}
				}

				text, err = r.ReadBytes('\n')
				if err == io.EOF {
					if len(found.occurrences) == 0 {
						item.Close()
					} else {
						out <- found
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
