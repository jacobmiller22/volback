package zip

import (
	"archive/zip"
	"io"
	"os"
)

func ZipReader(r io.Reader, name string) (io.Reader, error) {

	pr, pw := io.Pipe()
	zw := zip.NewWriter(pw)

	cleanup := func(err error) {
		pw.CloseWithError(err)
	}

	go func() {
		// Capture errors from the zip-writing work.
		w, err := zw.Create(name)
		if err != nil {
			cleanup(err)
			return
		}

		if _, err = io.Copy(w, r); err != nil {
			cleanup(err)
			return
		}

		// Call Close on the zip writer.
		cleanup(zw.Close())
	}()

	return pr, nil
}

func ZipDir(path string) (io.Reader, error) {

	pr, pw := io.Pipe()
	zw := zip.NewWriter(pw)

	cleanup := func(err error) {
		pw.CloseWithError(err)
	}

	go func() {
		// Capture errors from the zip-writing work.
		if err := zw.AddFS(os.DirFS(path)); err != nil {
			cleanup(err)
			return
		}

		// Call Close on the zip writer.
		if err := zw.Close(); err != nil {
			cleanup(err)
			return
		}

		// Now explicitly close the pipe writer.
		pw.Close()
	}()

	return pr, nil
}
