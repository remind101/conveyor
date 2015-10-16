package conveyor

import (
	"io/ioutil"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor/builder"
)

func TestWorker(t *testing.T) {
	l := builder.NewLogger(ioutil.Discard)
	b := new(mockBuilder)
	f := func(options builder.BuildOptions) (builder.Logger, error) {
		return l, nil
	}
	q := make(chan BuildRequest, 1)
	w := &Worker{
		Builder:       b,
		LogFactory:    f,
		buildRequests: q,
	}

	done := make(chan struct{})
	go func() {
		w.Start()
		close(done)
	}()

	b.On("Build", l, builder.BuildOptions{}).Return("", nil)

	q <- BuildRequest{
		Ctx:          context.Background(),
		BuildOptions: builder.BuildOptions{},
	}
	close(q)

	<-done
}
