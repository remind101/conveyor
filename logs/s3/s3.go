package s3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Logs returns a builder.Logs implementation that reads and writes logs to s3
// files.
type Logs struct {
	// Bucket that the log files will be stored in.
	Bucket string

	client *s3.S3
}

func NewLogger(bucket string) *Logs {
	c := s3.New(defaults.DefaultConfig)

	return &Logs{
		Bucket: bucket,
		client: c,
	}
}

func (l *Logs) Create(name string) (io.Writer, error) {
	name = filepath.Join("logs", fmt.Sprintf("%s.txt", name))

	return &writer{
		bucket: l.Bucket,
		name:   name,
		client: l.client,
		b:      new(bytes.Buffer),
	}, nil
}

func (l *Logs) Open(name string) (io.Reader, error) {
	return strings.NewReader(""), errors.New("s3 logs: read is not implemented yet")
}

// writer is an io.WriteCloser implementation that buffers up the bytes until
// Close is called, then flushes the data to a file in s3.
type writer struct {
	// Data will be buffered here.
	b *bytes.Buffer

	bucket, name string
	client       *s3.S3
}

func (l *writer) Write(p []byte) (int, error) {
	return l.b.Write(p)
}

func (l *writer) Close() error {
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
