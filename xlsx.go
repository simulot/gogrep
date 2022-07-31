package main

import (
	"io/fs"
	"strings"
	"sync/atomic"

	"github.com/xuri/excelize/v2"
)

func (a *App) ProcessXlsxFile(fsys fs.FS, name, archive string) error {
	atomic.AddInt64(&a.filesParsed, 1)
	f, err := fsys.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	wb, err := excelize.OpenReader(a.CountReader(f))
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
				if !a.stringExpSearch {
					loc := a.regexp.FindStringIndex(cell)
					if loc == nil {
						continue
					}
					grid, _ := excelize.CoordinatesToCellName(c+1, r+1)
					a.OutputHit(Hit{
						Archive:    archive,
						File:       name + "!" + sheet + "[" + grid + "]",
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
					File:       name + "!" + sheet + "[" + grid + "]",
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
