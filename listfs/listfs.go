package listfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type ListFS struct {
	files chan item
}

type item struct {
	fsys fs.FS
	name string
	err  error
}

func Open(params []string) (*ListFS, error) {
	files := make(chan item)
	lfs := &ListFS{files: files}
	go lfs.run(params)
	return lfs, nil
}

func (lfs *ListFS) run(params []string) {
	defer func() {
		close(lfs.files)
	}()
	for _, p := range params {
		s, err := os.Stat(p)
		if err != nil {
			fs, err := filepath.Glob(p)
			if err != nil {
				lfs.files <- item{nil, "", err}
				return
			}
			for _, f := range fs {
				lfs.files <- item{os.DirFS(filepath.Dir(f)), filepath.Base(f), nil}
			}
			continue
		}
		if !s.IsDir() {
			lfs.files <- item{os.DirFS(filepath.Dir(p)), filepath.Base(p), nil}
			continue
		} else {
			fsys := os.DirFS(p)
			err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				lfs.files <- item{fsys, path, nil}
				return nil
			})
			if err != nil {
				lfs.files <- item{nil, "", err}
				return
			}
			continue
		}
	}
	return
}

func (lfs *ListFS) Next() (fsys fs.FS, n string, err error) {
	var ok bool
	m := true
	var item item
	for m {
		item, ok = <-lfs.files
		if !ok {
			err = io.EOF
			return
		}
		fsys, n, err = item.fsys, item.name, item.err
		if item.err != nil || !m {
			return nil, "", item.err
		}
		return
	}
	return nil, "", io.EOF
}
