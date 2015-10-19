package conveyor

import (
	"io"

	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

// mockBuilder is a mock implementation of the builder.Builder interface.
type mockBuilder struct {
	mock.Mock
}

func (b *mockBuilder) Build(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
	args := b.Called(w, options)
	return args.String(0), args.Error(1)
}

// mockCancelBuilder is a mockBuilder that responds to Cancel.
type mockCancelBuilder struct {
	mockBuilder
}

func (b *mockCancelBuilder) Cancel() error {
	args := b.Called()
	return args.Error(0)
}

// mockBuildQueue is an implementation of the BuildQueue interface for testing.
type mockBuildQueue struct {
	mock.Mock
}

func (q *mockBuildQueue) Push(ctx context.Context, options builder.BuildOptions) error {
	args := q.Called(options)
	return args.Error(0)
}

func (q *mockBuildQueue) Subscribe(ch chan BuildRequest) error {
	args := q.Called(ch)
	return args.Error(0)
}
