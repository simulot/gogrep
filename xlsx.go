package main

import (
	"io"
	"strings"
	"sync/atomic"

	"github.com/xuri/excelize/v2"
)

func (a *App) ProcessXlsxFile(r io.Reader, name string) error {
	atomic.AddInt64(&a.filesParsed, 1)
	wb, err := excelize.OpenReader(a.CountReader(r))
	if err != nil {
		return err
	}

	for _, sheet := range wb.GetSheetList() {
		rows, err := wb.GetRows(sheet)
		if err != nil {
			return err
		}
		for r, cells := range rows {
			for c, cell := range cells {
				if a.regEpxSearch {
					loc := a.regexp.FindStringIndex(cell)
					if loc == nil {
						continue
					}
					grid, _ := excelize.CoordinatesToCellName(c+1, r+1)
					a.OutputHit(Hit{
						Archive:    name,
						File:       sheet + "[" + grid + "]",
						LineNumber: r,
						Line:       cell,
						Loc:        loc,
					})
					continue
				}

				i := strings.Index(cell, a.string)
				if i < 0 {
					continue
				}
				grid, _ := excelize.CoordinatesToCellName(c+1, r+1)
				a.OutputHit(Hit{
					Archive:    name,
					File:       sheet + "[" + grid + "]",
					LineNumber: r,
					Line:       cell,
					Loc:        []int{i, i + len(a.string)},
				})
				continue
			}
		}
	}
	return nil
}
