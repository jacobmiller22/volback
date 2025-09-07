package pipes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var (
	ErrProcessing error = errors.New("processing")
	ErrConfig     error = errors.New("misconfiguration")
)

type IOTransformFunc func(ctx context.Context, input io.Reader, output io.Writer) error

type IOPipe struct {
	name      string
	transform IOTransformFunc
}

func NewIOPipe(name string, transform IOTransformFunc) IOPipe {
	return IOPipe{
		name,
		transform,
	}
}

// Process this Pipe
func (p *IOPipe) Process(ctx context.Context, r io.Reader, w io.Writer) error {
	slog.DebugContext(ctx, "process_start", "name", p.name)

	if err := p.transform(ctx, r, w); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrProcessing, p.name, err)
	}

	slog.DebugContext(ctx, "process_finish", "name", p.name)
	return nil
}

type IOPipeline struct {
	pipes []IOPipe
}

// Creates a new IOPipeline
func NewIOPipeline(pipes []IOPipe) (*IOPipeline, error) {

	for i, pipe := range pipes {
		if pipe.transform == nil {
			return nil, fmt.Errorf("%w: pipe at index %d has nil transform", ErrConfig, i)
		}
	}

	return &IOPipeline{
		pipes,
	}, nil
}

// IOPipeline.Execute configures multiple IO stages together to form a concurrent processing
// pipeline where input is taken from the given io.reader, and output can be read from the
// returned io.Reader
// Processing errors get propogated to the .CloseWithError() calls.
func (pl *IOPipeline) Execute(ctx context.Context, r io.Reader) io.Reader {
	for _, pipe := range pl.pipes {
		pr, pw := io.Pipe()

		go func(p IOPipe, reader io.Reader, writer *io.PipeWriter) {
			if closer, ok := reader.(io.Closer); ok {
				defer closer.Close()
			}

			err := p.Process(ctx, reader, writer)

			defer writer.CloseWithError(err)
			if err != nil {
				slog.DebugContext(ctx, "process_error", "name", p.name, "error", err)
			}
		}(pipe, r, pw)

		r = pr
	}

	return r
}
