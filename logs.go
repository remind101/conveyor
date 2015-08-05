package conveyor

import (
	"fmt"
	"io"
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
		name := filepath.Join("logs", opts.Repository, fmt.Sprintf("%s-%s", opts.Sha, uuid.New()))
		return b.PutWriter(name, nil, nil)
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

		return io.MultiWriter(writers...), nil
	}
}
