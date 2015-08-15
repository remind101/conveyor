package conveyor

import (
	"errors"
	"io"
	"testing"

	"github.com/remind101/conveyor/builder"

	"golang.org/x/net/context"
)

func TestConveyor_Build(t *testing.T) {
	b := func(ctx context.Context, w io.Writer, opts builder.BuildOptions) (string, error) {
		return "", nil
	}
	w := &closeWriter{}
	c := New(builder.BuilderFunc(b))

	if _, err := c.Build(context.Background(), w, builder.BuildOptions{}); err != nil {
		t.Fatal(err)
	}

	if !w.closed {
		t.Fatal("Expected logger to be closed")
	}
}

func TestConveyor_Build_CloseError(t *testing.T) {
	closeErr := errors.New("i/o timeout")
	b := func(ctx context.Context, w io.Writer, opts builder.BuildOptions) (string, error) {
		return "", nil
	}
	w := &closeWriter{closeErr: closeErr}
	c := New(builder.BuilderFunc(b))

	if _, err := c.Build(context.Background(), w, builder.BuildOptions{}); err != closeErr {
		t.Fatalf("Expected error to be %v", closeErr)
	}

	if !w.closed {
		t.Fatal("Expected logger to be closed")
	}
}

type closeWriter struct {
	closeErr error
	closed   bool
}

func (w *closeWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *closeWriter) Close() error {
	w.closed = true
	return w.closeErr
}
