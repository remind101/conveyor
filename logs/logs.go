// Package logs provides an interface and implementations for reading and
// writing streaming logs.
package logs

import (
	"io"
	"io/ioutil"
	"strings"
)

// Discard is a Logger that returns noop io.Reader and io.Writers.
var Discard = &nullLogger{}

type Logger interface {
	// Create returns an io.Writer that can be written to.
	Create(name string) (io.Writer, error)

	// Open returns an io.Reader that can be read from to stream the logs
	// back to the client.
	Open(name string) (io.Reader, error)
}

// nullLogger is a BuildLogs implementation that returns null readers and
// writers.
type nullLogger struct{}

func (l *nullLogger) Create(name string) (io.Writer, error) {
	return ioutil.Discard, nil
}

func (l *nullLogger) Open(name string) (io.Reader, error) {
	return strings.NewReader(""), nil
}
