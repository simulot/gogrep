package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"fmt"

	mytgzfs "github.com/simulot/gogrep/mytarfs"
	"github.com/simulot/gogrep/myzipfs"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type Nexter interface {
	Next() (fs.FS, string, error)
}

func (a *App) IsMatch(f string) bool {
	m, _ := filepath.Match(a.mask, filepath.Base(f))
	return m
}

func (a *App) ProcessArchive(ctx context.Context, n Nexter, archive string) error {
	for {
		fsys, name, err := n.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = a.ProcessAnyFile(ctx, fsys, name, archive)
		if err != nil {
			return err
		}
	}
}

// Process any file to apply the appropriate treatment for zip, xlsx, tar, tgz files, and handling char set for text files
func (a *App) ProcessAnyFile(ctx context.Context, fsys fs.FS, name string, archive string) error {

	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".gsheet", ".gslides", ".gdoc", ".eps":
		return nil // Discard gsuite files that would need a special treatment
	case ".xlsx":
		return a.ProcessXlsxFile(fsys, name, archive)
	case ".tgz":
		return a.ProcessTGZ(ctx, fsys, name, archive)
	case ".zip":
		return a.ProcessZipFile(ctx, fsys, name, archive)
	}
	if len(a.mask) > 0 {
		if !a.IsMatch(filepath.Base(name)) {
			return nil
		}
	}
	a.group.Go(func() error { return a.ProcessTextFile(ctx, fsys, name, archive) })
	return nil
}

// ProcessTextFiles opens the file, determine the charset, and uses the correct decoder
func (a *App) ProcessTextFile(ctx context.Context, fsys fs.FS, name string, archive string) error {
	// regular files
	sniff := make([]byte, 512)

	f, err := fsys.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Read(sniff)
	if err != nil && err != io.EOF {
		return err
	}

	mr := io.MultiReader(bytes.NewReader(sniff), f)
	t := http.DetectContentType(sniff)
	switch {
	case t == "text/plain; charset=utf-16be":
		return a.ProcessUTF16be(ctx, mr, name, archive)
	case t == "text/plain; charset=utf-16le":
		return a.ProcessUTF16le(ctx, mr, name, archive)
	case t == "text/plain; charset=utf-8" || t == "text/xml; charset=utf-8" || t == "application/octet-stream":
		return a.ProcessUTF8(ctx, mr, name, archive)
	}
	return nil
}

// ProcessUTF16be convert UTF16be file into UTF8
func (a *App) ProcessUTF16be(ctx context.Context, r io.Reader, name, archive string) error {
	r = transform.NewReader(r, unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder())
	return a.ProcessUTF8(ctx, r, name, archive)
}

// ProcessUTF16le convert UTF16le file into UTF8
func (a *App) ProcessUTF16le(ctx context.Context, r io.Reader, name, archive string) error {
	r = transform.NewReader(r, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	return a.ProcessUTF8(ctx, r, name, archive)
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 512*1024)
	},
}

// ProcessUTF8 for ascii and utf8 files
func (a *App) ProcessUTF8(ctx context.Context, r io.Reader, name string, archive string) error {
	defer atomic.AddInt64(&a.filesParsed, 1)
	br := bufio.NewReader(a.CountReader(r))
	line := 0
	var err error
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s, err := br.ReadString('\n')
			if err != nil {
				break loop
			}
			line++
			loc := a.regexp.FindStringIndex(s)
			if loc == nil {
				continue
			}
			a.OutputHit(Hit{
				Archive:    archive,
				File:       name,
				LineNumber: line,
				Line:       s,
				Loc:        loc,
			})
		}
	}
	if err == io.EOF || err == nil {
		return nil
	}
	return fmt.Errorf("can't process '%s', at line %d, %w", name, line, err)
}

// readerAtFrom returns a ReaderAt from a fs.File by read all the stream if needed
func readerAtFrom(f fs.File) (io.ReaderAt, int64, string, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, "", err
	}
	osF, ok := f.(*os.File)
	if ok {
		return osF, fi.Size(), fi.Name(), nil
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, "", err
	}
	return bytes.NewReader(b), int64(len(b)), fi.Name(), nil
}

// ProcessZipFile open the archive and process each archived file
func (a *App) ProcessZipFile(ctx context.Context, fsys fs.FS, path string, archive string) error {
	f, err := fsys.Open(path)
	if err != nil {
		return err
	}

	s, err := f.Stat()
	if err != nil {
		return err
	}
	zfs, err := myzipfs.Reader(f, s.Size())
	if err != nil {
		return err
	}
	return a.ProcessArchive(ctx, zfs, path)
}

func (a *App) ProcessTGZ(ctx context.Context, fsys fs.FS, path string, archive string) error {
	f, err := fsys.Open(path)
	if err != nil {
		return err
	}
	t, err := mytgzfs.Reader(f, path)
	if err != nil {
		return err
	}
	return a.ProcessArchive(ctx, t, path)
}
