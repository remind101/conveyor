package worker

import (
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWorker(t *testing.T) {
	c := new(mockConveyor)
	b := new(mockBuilder)
	q := make(chan conveyor.BuildContext, 1)
	w := &Worker{
		Builder:       b,
		Conveyor:      c,
		buildRequests: q,
	}

	done := make(chan struct{})
	go func() {
		w.Start()
		close(done)
	}()

	b.On("Build", ioutil.Discard, builder.BuildOptions{
		ID: "1234",
	}).Return("remind101/acme-inc:abcd", nil)
	c.On("BuildStarted", "1234").Return(nil)
	c.On("BuildComplete", "1234", "remind101/acme-inc:abcd").Return(nil)

	q <- conveyor.BuildContext{
		Ctx: context.Background(),
		BuildOptions: builder.BuildOptions{
			ID: "1234",
		},
	}
	close(q)

	<-done
}

func TestWorker_Shutdown(t *testing.T) {
	c := new(mockConveyor)
	b := new(mockBuilder)
	q := make(chan conveyor.BuildContext, 1)
	w := &Worker{
		Builder:       b,
		Conveyor:      c,
		buildRequests: q,
		shutdown:      make(chan struct{}),
		done:          make(chan error),
	}

	done := make(chan struct{})
	go func() {
		w.Start()
		close(done)
	}()

	err := w.Shutdown()

	<-done

	assert.NoError(t, err)
}

func TestWorker_Shutdown_Cancel(t *testing.T) {
	c := new(mockConveyor)
	b := new(mockCancelBuilder)
	q := make(chan conveyor.BuildContext, 1)
	w := &Worker{
		Builder:       b,
		Conveyor:      c,
		buildRequests: q,
		shutdown:      make(chan struct{}),
		done:          make(chan error),
	}

	done := make(chan struct{})
	go func() {
		w.Start()
		close(done)
	}()

	b.On("Cancel").Return(nil)
	err := w.Shutdown()

	<-done

	assert.NoError(t, err)
}

func TestWorker_Shutdown_Cancel_Error(t *testing.T) {
	c := new(mockConveyor)
	b := new(mockCancelBuilder)
	q := make(chan conveyor.BuildContext, 1)
	w := &Worker{
		Builder:       b,
		Conveyor:      c,
		buildRequests: q,
		shutdown:      make(chan struct{}),
		done:          make(chan error),
	}

	done := make(chan struct{})
	go func() {
		w.Start()
		close(done)
	}()

	boom := errors.New("Failed to cancel")
	b.On("Cancel").Return(boom)
	err := w.Shutdown()

	<-done

	assert.Equal(t, boom, err)
}

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

type mockConveyor struct {
	mock.Mock
}

func (m *mockConveyor) Writer(ctx context.Context, buildID string) (io.Writer, error) {
	return ioutil.Discard, nil
}

func (m *mockConveyor) BuildStarted(ctx context.Context, buildID string) error {
	args := m.Called(buildID)
	return args.Error(0)
}

func (m *mockConveyor) BuildComplete(ctx context.Context, buildID string, image string) error {
	args := m.Called(buildID, image)
	return args.Error(0)
}

func (m *mockConveyor) BuildFailed(ctx context.Context, buildID string, err error) error {
	args := m.Called(buildID, err)
	return args.Error(0)
}
