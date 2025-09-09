package volback

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jacobmiller22/volume-backup/internal/zip"
)

type FsPushPuller struct {
	restore bool
}

// Pull pulls the given path and returns an io.Reader that will read
// zip-compressed contents if we are not restoring
func (p *FsPushPuller) Pull(path string) (io.Reader, error) {

	if p.restore {
		fd, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening path as file: %v", err)
		}
		return fd, nil
	}

	r, err := zip.CreateArchiveFromPath(path)
	if err != nil {
		return nil, fmt.Errorf("creating zip archive: %v", err)
	}
	return r, nil
}

// Push pushs the given reader to the path. If restore is true,
// the reader will be unpacked as a compress zip file to given the path
func (p *FsPushPuller) Push(r io.Reader, path string) error {

	if p.restore {
		return zip.UnpackArchiveToPath(r, path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	if _, err = io.Copy(fd, r); err != nil {
		return err
	}

	return nil
}
