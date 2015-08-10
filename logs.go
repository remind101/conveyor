package conveyor

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
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
func S3Logger(bucket string) (LogFactory, error) {
	c := s3.New(aws.DefaultConfig)

	return func(opts BuildOptions) (Logger, error) {
		name := filepath.Join("logs", opts.Repository, fmt.Sprintf("%s-%s.txt", opts.Sha, uuid.New()))

		r, w := io.Pipe()

		go func() {
			raw, err := ioutil.ReadAll(r)
			if err != nil {
				fmt.Printf("err: %v", err)
				return
			}

			if _, err := c.PutObject(&s3.PutObjectInput{
				Bucket:        aws.String(bucket),
				Key:           aws.String(name),
				ACL:           aws.String("public-read"),
				Body:          bytes.NewReader(raw),
				ContentLength: aws.Int64(int64(len(raw))),
				ContentType:   aws.String("text/plain"),
			}); err != nil {
				fmt.Printf("err: %v", err)
				return
			}
		}()

		return &logger{
			WriteCloser: w,
			url:         fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, name),
		}, nil
	}, nil
}
