package conveyor

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"

	"github.com/rlmcpherson/s3gof3r"
)

// Logger is a place where logs are written to.
type Logger interface {
	// Loggers implement Write and Close methods.
	io.WriteCloser

	// URL should return the URL to view the logs.
	URL() string
}

type logger struct {
	io.WriteCloser
	url string
}

func (l *logger) URL() string {
	return l.url
}

type stdoutLogger struct{}

func (l *stdoutLogger) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (l *stdoutLogger) Close() error                { return nil }
func (l *stdoutLogger) URL() string                 { return "" }

// LogFactory is a function that can return a location to write logs to for a
// build.
type LogFactory func(BuildOptions) (Logger, error)

func StdoutLogger(opts BuildOptions) (Logger, error) {
	return &stdoutLogger{}, nil
}

// S3Logger returns a log factory that writes logs to a file in an S3
// bucket.
func S3Logger(bucket string, keys func() (s3gof3r.Keys, error)) (LogFactory, error) {
	k, err := keys()
	if err != nil {
		return nil, err
	}

	b := s3gof3r.New("", k).Bucket(bucket)
	return func(opts BuildOptions) (Logger, error) {
		name := filepath.Join("logs", opts.Repository, fmt.Sprintf("%s-%s.txt", opts.Sha, uuid.New()))
		h := make(http.Header)
		h.Add("Content-Type", "text/plain")
		w, err := b.PutWriter(name, h, nil)
		if err != nil {
			return nil, err
		}
		return &logger{
			WriteCloser: w,
			url:         fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, name),
		}, nil
	}, nil
}
