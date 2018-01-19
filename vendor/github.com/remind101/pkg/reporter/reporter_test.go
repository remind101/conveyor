package reporter

import (
	"errors"
	"net/http"
	"path"
	"runtime"
	"testing"

	"golang.org/x/net/context"
)

var errBoom = errors.New("boom")

func TestReport(t *testing.T) {
	r := ReporterFunc(func(ctx context.Context, err error) error {
		e := err.(*Error)

		if e.Request.Header.Get("Content-Type") != "application/json" {
			t.Fatal("request information not set")
		}

		line := e.Backtrace[0]
		fn := runtime.FuncForPC(line.PC)

		if got, want := path.Base(fn.Name()), "reporter.TestReport"; got != want {
			t.Fatalf("expected the first backtrace line to be this function")
		}

		return nil
	})
	ctx := WithReporter(context.Background(), r)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Type", "application/json")
	AddRequest(ctx, req)

	if err := Report(ctx, errBoom); err != nil {
		t.Fatal(err)
	}
}

func TestMonitor(t *testing.T) {
	ensureRepanicked := func() {
		if v := recover(); v == nil {
			t.Errorf("Must have panicked after reporting!")
		}
	}
	var reportedError error
	r := ReporterFunc(func(ctx context.Context, err error) error {
		reportedError = err
		return nil
	})
	ctx := WithReporter(context.Background(), r)

	done := make(chan interface{})
	go func() {
		defer close(done)
		defer ensureRepanicked()
		defer Monitor(ctx)
		panic("oh noes!")
	}()
	<-done
	if reportedError == nil || reportedError.Error() != "panic: oh noes!" {
		t.Errorf("expected panic 'oh noes!' to be reported, got %#v", reportedError)
	}
}
