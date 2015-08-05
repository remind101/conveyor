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

// LogFactory is a function that can return a location to write logs to for a
// build.
type LogFactory func(BuildOptions) (io.Writer, error)

func StdoutLogger(opts BuildOptions) (io.Writer, error) {
	return os.Stdout, nil
}

// S3Logger returns a log factory that writes logs to a file in an S3
// bucket.
func S3Logger(bucket string, keys func() (s3gof3r.Keys, error)) (LogFactory, error) {
	k, err := keys()
	if err != nil {
		return nil, err
	}

	b := s3gof3r.New("", k).Bucket(bucket)
	return func(opts BuildOptions) (io.Writer, error) {
		name := filepath.Join("logs", opts.Repository, fmt.Sprintf("%s-%s.txt", opts.Sha, uuid.New()))
		h := make(http.Header)
		h.Add("Content-Type", "text/plain")
		return b.PutWriter(name, h, nil)
	}, nil
}

// MultiLogger is a LogFactory that writes to multiple logs.
func MultiLogger(f ...LogFactory) LogFactory {
	return func(opts BuildOptions) (io.Writer, error) {
		var writers []io.Writer

		for _, ff := range f {
			w, err := ff(opts)
			if err != nil {
				return nil, err
			}
			writers = append(writers, w)
		}

		return MultiWriteCloser(writers...), nil
	}
}

type multiWriteCloser struct {
	writers []io.Writer
	io.Writer
}

func MultiWriteCloser(writers ...io.Writer) io.WriteCloser {
	return &multiWriteCloser{
		Writer:  io.MultiWriter(writers...),
		writers: writers,
	}
}

func (t *multiWriteCloser) Close() error {
	var errors []error
	for _, w := range t.writers {
		if w, ok := w.(io.Closer); ok {
			if err := w.Close(); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}
