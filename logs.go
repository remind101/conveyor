package conveyor

import (
	"bytes"
	"fmt"
	"io"
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

// LogFactory is a function that can return a location to write logs to for a
// build.
type LogFactory func(BuildOptions) (Logger, error)

func StdoutLogger(opts BuildOptions) (Logger, error) {
	return &stdoutLogger{}, nil
}

type stdoutLogger struct{}

func (l *stdoutLogger) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (l *stdoutLogger) Close() error                { return nil }
func (l *stdoutLogger) URL() string                 { return "" }

// NullLogger is a logger that does nothing.
type NullLogger struct{}

func (l *NullLogger) Write(p []byte) (int, error) { return len(p), nil }
func (l *NullLogger) Close() error                { return nil }
func (l *NullLogger) URL() string                 { return "" }

// S3Logger returns a log factory that writes logs to a file in an S3
// bucket.
func S3Logger(bucket string) (LogFactory, error) {
	c := s3.New(aws.DefaultConfig)

	return func(opts BuildOptions) (Logger, error) {
		name := filepath.Join("logs", opts.Repository, fmt.Sprintf("%s-%s.txt", opts.Sha, uuid.New()))

		return &s3Logger{
			bucket: bucket,
			name:   name,
			client: c,
			b:      new(bytes.Buffer),
		}, nil
	}, nil
}

type s3Logger struct {
	// Data will be buffered here.
	b *bytes.Buffer

	bucket, name string
	client       *s3.S3
}

func (l *s3Logger) Write(p []byte) (int, error) {
	return l.b.Write(p)
}

func (l *s3Logger) Close() error {
	_, err := l.client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(l.bucket),
		Key:           aws.String(l.name),
		ACL:           aws.String("public-read"),
		Body:          bytes.NewReader(l.b.Bytes()),
		ContentLength: aws.Int64(int64(l.b.Len())),
		ContentType:   aws.String("text/plain"),
	})
	return err
}

func (l *s3Logger) URL() string {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", l.bucket, l.name)
}
