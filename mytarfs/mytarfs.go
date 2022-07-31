package mytgzfs

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
)

/*

mytarfs implements a sort of fs.FS interface for tgz files


*/

type TgzFs struct {
	*tar.Reader
	current *tar.Header
}

func Open(fsys fs.FS, name string) (*TgzFs, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	return Reader(f, name)
}

func Reader(r io.Reader, name string) (*TgzFs, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(zr)
	return &TgzFs{
		Reader: tr,
	}, nil
}

// Implement fs.FS. Next opens the next file in the archive that match the mask.
// It returns a fs.File, or io.EOF when the end of the archive is reached.

func (tgz *TgzFs) Next() (fs.FS, string, error) {
	var h *tar.Header
	var err error
	for {
		h, err = tgz.Reader.Next()
		if err != nil {
			return nil, "", err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		break
	}
	tgz.current = h
	return tgz, h.Name, nil

}

func (tgz *TgzFs) Open(name string) (fs.File, error) {
	return &TgzFile{
		Reader: tgz,
		info:   tgz.current.FileInfo(),
	}, nil

}

type TgzFile struct {
	io.Reader
	info fs.FileInfo
}

func (tgzf *TgzFile) Stat() (fs.FileInfo, error) {
	return tgzf.info, nil
}

func (tgzf *TgzFile) Read(b []byte) (int, error) {
	return tgzf.Reader.Read(b)
}

func (tgzf *TgzFile) Close() error {
	return nil
}
