package builder

import (
	"io"
	"io/ioutil"
	"strings"
)

// Logs represents an interface for obtaining readers and writers for build
// logs.
type Logs interface {
	// Writer creates an io.Writer where log output can be written to.
	Writer(id string) (io.Writer, error)

	// Reader returns an io.Reader that streams the build output for the
	// given build with the given id.
	Reader(id string) (io.Reader, error)
}

var DiscardLogs = &nullLogs{}

// nullLogs is a BuildLogs implementation that returns null readers and
// writers.
type nullLogs struct{}

func (l *nullLogs) Writer(id string) (io.Writer, error) {
	return ioutil.Discard, nil
}

func (l *nullLogs) Reader(id string) (io.Reader, error) {
	return strings.NewReader(""), nil
}
