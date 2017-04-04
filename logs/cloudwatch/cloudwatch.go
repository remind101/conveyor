package cloudwatch

import (
	"io"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"
)

func NewLogger(config client.ConfigProvider, group string) *Group {
	c := cloudwatchlogs.New(config)
	return &Group{cloudwatch.NewGroup(group, c)}
}

type Group struct {
	*cloudwatch.Group
}

func (g *Group) Create(name string) (io.Writer, error) {
	w, err := g.Group.Create(name)
	if err != nil {
		return w, err
	}
	return &writer{w.(io.WriteCloser)}, nil
}

func (g *Group) Open(name string) (io.Reader, error) {
	r, err := g.Group.Open(name)
	if err != nil {
		return r, err
	}
	return &reader{Reader: r}, nil
}

// http://www.nthelp.com/ascii.htm
const endOfText = '\x03'

// writer is an io.WriteCloser that writes the endOfText token when the stream is
// closed.
type writer struct {
	io.WriteCloser
}

func (w *writer) Close() error {
	_, err := w.Write([]byte{endOfText})
	if err != nil {
		return err
	}
	return w.WriteCloser.Close()
}

// reader is an io.Reader that reads until the endOfText token is read.
type reader struct {
	io.Reader
	closed bool
}

func (r *reader) Read(b []byte) (int, error) {
	if r.closed == true {
		return 0, io.EOF
	}

	n, err := r.Reader.Read(b)
	if err != nil {
		return n, err
	}

	if n > 0 && b[n-1] == endOfText {
		r.closed = true
		return n, io.EOF
	}

	return n, nil
}
