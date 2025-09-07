package volback

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
)

type FsPushPuller struct{}

func (p *FsPushPuller) Pull(path string) (io.Reader, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func (p *FsPushPuller) Push(r io.Reader, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Reader from r and push to the path specified
	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	w := bufio.NewWriter(fd)

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	return w.Flush()
}
