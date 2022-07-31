package myzipfs

/*
	myzipfs implements fs.FS and fs.ReadDirFS interface for a zip archive.

*/

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"path/filepath"
)

type ZipFs struct {
	*zip.Reader
	index int
}

// Open a zip file for the fs. If the file can't provide a io.ReadAt, the file is
// completely read in a buffer acting as a io.ReadAt.
func Open(fsys fs.FS, name string) (*ZipFs, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return Reader(f, s.Size())
}

// makeReaderAt returns a ReaderAt from an io.Reader using ReadAll when needed
func makeReaderAt(r io.Reader) (io.ReaderAt, error) {
	if ra, ok := r.(io.ReaderAt); ok {
		return ra, nil
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// Reader return a ZipFs
func Reader(r io.Reader, size int64) (*ZipFs, error) {
	var err error
	zfs := ZipFs{}
	ra, err := makeReaderAt(r)
	if err != nil {
		return nil, err
	}
	zfs.Reader, err = zip.NewReader(ra, size)
	if err != nil {
		return nil, err
	}
	return &zfs, nil
}

// ReadDir implements the ReadDir interface
func (zfs *ZipFs) ReadDir(name string) ([]fs.DirEntry, error) {
	ds := []fs.DirEntry{}
	for _, d := range zfs.File {
		ds = append(ds, &DirEntry{File: d})
	}
	return ds, nil
}

// DirEntry adds missing methods to implement fs.FileInfo
type DirEntry struct {
	*zip.File
}

func (de DirEntry) Info() (fs.FileInfo, error) {
	return de.FileInfo(), nil
}
func (de DirEntry) IsDir() bool {
	return de.Mode().IsDir()
}

func (de DirEntry) Name() string {
	return de.File.Name
}

func (de DirEntry) Type() fs.FileMode {
	return de.Mode()
}

func (zfs *ZipFs) Next(mask string) (fs.File, string, error) {
	for ; zfs.index < len(zfs.Reader.File); zfs.index++ {
		f := zfs.Reader.File[zfs.index]
		zfs.index++
		if f.FileInfo().IsDir() {
			continue
		}
		m, _ := filepath.Match(mask, filepath.Base(f.Name))
		if !m {
			continue
		}
		r, err := zfs.Reader.Open(f.Name)
		return r, f.Name, err
	}
	return nil, "", io.EOF
}
