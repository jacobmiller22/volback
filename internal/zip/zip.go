package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateArchiveFromPath returns a *zip.Reader from a filesystem path
func CreateArchiveFromPath(path string) (io.Reader, error) {

	pinfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat(%s): %v", path, err)
	}

	pr, pw := io.Pipe()

	if pinfo.IsDir() {
		go func(pw *io.PipeWriter) {
			zw := zip.NewWriter(pw)

			// th, err := zip.FileInfoHeader(pinfo, "")
			// if err != nil {
			// 	pw.CloseWithError(fmt.Errorf("zip header creation: %v", err))
			// 	return
			// }
			// th.Name = filepath.Base(path)

			// if err := zw.WriteHeader(th); err != nil {
			// 	pw.CloseWithError(fmt.Errorf("writing header to zip: %v", err))
			// 	return
			// }

			zw.AddFS(os.DirFS(path))

			zw.Close()
			pw.Close()
		}(pw)
	} else {
		fd, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("Open(%s): %v", path, err)
		}

		go func(r io.Reader, pw *io.PipeWriter) {
			zw := zip.NewWriter(pw)

			zh, err := zip.FileInfoHeader(pinfo)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("zip header creation: %v", err))
				return
			}
			zh.Name = filepath.Base(path)

			fw, err := zw.CreateHeader(zh)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("writing header to zip: %v", err))
				return
			}

			if _, err := io.Copy(fw, r); err != nil {
				pw.CloseWithError(fmt.Errorf("writing contents to zip: %v", err))
				return
			}

			zw.Close()
			pw.Close()
		}(fd, pw)
	}

	return pr, nil
}

func UnpackArchiveToPath(r io.Reader, extractPath string) error {

	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return err
	}

	buf, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		return err
	}

	for _, zf := range zr.File {
		path := filepath.Join(extractPath, zf.Name)

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(path, zf.Mode()); err != nil {
				return err
			}
			continue
		}

		// Open the file in the archive
		rc, err := zf.Open()
		if err != nil {

			continue
		}
		defer rc.Close() // Ensure the file is closed

		// Create the output file
		outFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer outFile.Close() // Ensure the output file is closed

		// Copy the content
		if _, err = io.Copy(outFile, rc); err != nil {
			return err
		}

	}

	return nil
}
