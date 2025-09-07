package pipes

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func passthroughTransformer(ctx context.Context, input io.Reader, output io.Writer) error {
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return nil
}

func TestPipelineExecute(t *testing.T) {
	given := "Hello world that is larger than the 16 byte buffer"
	initialReader := strings.NewReader(given)

	pl, err := NewIOPipeline(
		[]IOPipe{
			NewIOPipe("1", passthroughTransformer),
			NewIOPipe("2", passthroughTransformer),
			NewIOPipe("3", passthroughTransformer),
			NewIOPipe("4", passthroughTransformer),
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := pl.Execute(t.Context(), initialReader)

	got, err := io.ReadAll(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diff := cmp.Diff(given, string(got)); diff != "" {
		t.Fatal(diff)
	}
}
