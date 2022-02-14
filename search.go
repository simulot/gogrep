package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"fmt"

	"golang.org/x/sync/errgroup"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func (a *App) ProcessFile(f fs.File, archive string) error {
	sniff := make([]byte, 512)

	fInfo, err := f.Stat()
	if err != nil {
		return err
	}
	if fInfo.Size() == 0 {
		return nil
	}
	name := fInfo.Name()
	_, err = f.Read(sniff)
	if err != nil && err != io.EOF {
		return err
	}

	mr := io.MultiReader(bytes.NewReader(sniff), f)
	t := http.DetectContentType(sniff)
	switch t {
	case "application/zip":
		return a.ProcessZipFile(mr, f)
	case "application/x-gzip":
		return a.ProcessTGZ(mr, f)
	// case "application/x-rar-compressed"
	// case "application/x-rar-compressed"
	case "text/plain; charset=utf-16be":
		return a.ProcessUTF16be(mr, name, archive)
	case "text/plain; charset=utf-16le":
		return a.ProcessUTF16le(mr, name, archive)
	case "text/plain; charset=utf-8":
		return a.ProcessUTF8(mr, name, archive)
	default:
		fmt.Println("Skiping", name, t)
	}

	return nil
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 512*1024)
	},
}

func (a *App) ProcessUTF8(r io.Reader, name string, archive string) error {
	atomic.AddInt64(&a.filesParsed, 1)
	s := bufio.NewScanner(a.CountReader(r))
	buffer := bufferPool.Get().([]byte)
	defer bufferPool.Put(buffer)
	s.Buffer(buffer, 512*1024)
	line := 0
	for s.Scan() {
		line++
		if a.regEpxSearch {
			loc := a.regexp.FindIndex(s.Bytes())
			if loc == nil {
				continue
			}
			a.OutputHit(Hit{
				Archive:    archive,
				File:       name,
				LineNumber: line,
				Line:       s.Text(),
				Loc:        loc,
			})
			continue
		}

		i := bytes.Index(s.Bytes(), []byte(a.string))
		if i < 0 {
			continue
		}
		a.OutputHit(Hit{
			Archive:    archive,
			File:       name,
			LineNumber: line,
			Line:       s.Text(),
			Loc:        []int{i, i + len(a.string)},
		})
		continue
	}
	if s.Err() != nil {
		return fmt.Errorf("can't process '%s', at line %d, %w", name, line, s.Err())
	}
	return nil
}

func (a *App) ProcessUTF16be(r io.Reader, name, archive string) error {
	r = transform.NewReader(r, unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder())
	return a.ProcessUTF8(r, name, archive)
}
func (a *App) ProcessUTF16le(r io.Reader, name, archive string) error {
	r = transform.NewReader(r, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	return a.ProcessUTF8(r, name, archive)
}

func readerAtFrom(r io.Reader, f fs.File) (io.ReaderAt, int64, string, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, "", err
	}
	osF, ok := f.(*os.File)
	if ok {
		return osF, fi.Size(), fi.Name(), nil
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, "", err
	}
	return bytes.NewReader(b), int64(len(b)), fi.Name(), nil
}

func (a *App) ProcessZipFile(r io.Reader, archive fs.File) error {
	readerAt, size, archiveName, err := readerAtFrom(r, archive)
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(readerAt, size)
	if err != nil {
		return err
	}

	group := errgroup.Group{}

	for _, zipEntry := range zipReader.File {
		if !zipEntry.FileInfo().IsDir() && zipEntry.FileInfo().Size() > 0 {
			if len(a.mask) > 0 && !a.IsMatch(zipEntry.Name) {
				continue
			}
			zipEntry := zipEntry
			a.limiter.Start()
			group.Go(func() error {
				defer a.limiter.Done()
				f, err := zipEntry.Open()
				if err != nil {
					return fmt.Errorf("can't process file '%s' from archive '%s', %w", zipEntry.Name, archiveName, err)
				}
				defer f.Close()
				fs := f.(fs.File)
				err = a.ProcessFile(fs, archiveName)
				if err != nil {
					return fmt.Errorf("can't process file '%s' from archive '%s', %w", zipEntry.Name, archiveName, err)
				}
				return err
			})
		}
	}
	return group.Wait()
}

type tarFile struct {
	*tar.Header
	*tar.Reader
}

func (tf *tarFile) Stat() (fs.FileInfo, error) {
	return tf.FileInfo(), nil
}
func (tf *tarFile) Close() error {
	return nil
}

func (a *App) ProcessTGZ(r io.Reader, archive fs.File) error {
	fi, err := archive.Stat()
	if err != nil {
		return err
	}
	zipReader, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	// group := errgroup.Group{}
	tarReader := tar.NewReader(zipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("can't decompress tgz file %s, %w", fi.Name(), err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if len(a.mask) > 0 && !a.IsMatch(header.Name) {
			io.Copy(ioutil.Discard, tarReader)
			continue
		}
		file := &tarFile{
			Header: header,
			Reader: tarReader,
		}
		err = a.ProcessFile(file, fi.Name())
		if err != nil {
			return fmt.Errorf("can't process file '%s' from archive '%s', %w", header.Name, fi.Name(), err)
		}
	}
	// return group.Wait()
	return err
}
