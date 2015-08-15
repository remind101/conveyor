package conveyor

import (
	"errors"
	"testing"

	"github.com/remind101/conveyor/builder"

	"golang.org/x/net/context"
)

func TestConveyor_Build(t *testing.T) {
	b := func(ctx context.Context, w builder.Logger, opts builder.BuildOptions) (string, error) {
		return "", nil
	}
	w := &mockLogger{}
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
	b := func(ctx context.Context, w builder.Logger, opts builder.BuildOptions) (string, error) {
		return "", nil
	}
	w := &mockLogger{closeErr: closeErr}
	c := New(builder.BuilderFunc(b))

	if _, err := c.Build(context.Background(), w, builder.BuildOptions{}); err != closeErr {
		t.Fatalf("Expected error to be %v", closeErr)
	}

	if !w.closed {
		t.Fatal("Expected logger to be closed")
	}
}

type mockLogger struct {
	closeErr error
	closed   bool
}

func (m *mockLogger) Write(p []byte) (int, error) {
	return len(p), nil
}

func (m *mockLogger) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockLogger) URL() string {
	return "https://google.com"
}
