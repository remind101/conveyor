// Package logs provides an interface and implementations for reading and
// writing streaming logs.
package logs

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// Discard is a Logger that returns noop io.Reader and io.Writers.
var Discard = &nullLogger{}

// Stdout is a Logger that writes logs to os.Stdout.
var Stdout = &stdoutLogger{}

type Logger interface {
	// Create returns an io.Writer that can be written to.
	Create(name string) (io.Writer, error)

	// Open returns an io.Reader that can be read from to stream the logs
	// back to the client.
	Open(name string) (io.Reader, error)
}

// nullLogger is a Logger implementation that returns null readers and
// writers.
type nullLogger struct{}

func (l *nullLogger) Create(name string) (io.Writer, error) {
	return ioutil.Discard, nil
}

func (l *nullLogger) Open(name string) (io.Reader, error) {
	return strings.NewReader(""), nil
}

// stdLogger is a Logger implementation that writes log output to os.Stdout.
type stdoutLogger struct{}

func (l *stdoutLogger) Create(name string) (io.Writer, error) {
	return os.Stdout, nil
}

func (l *stdoutLogger) Open(name string) (io.Reader, error) {
	return strings.NewReader(""), errors.New("stdout logger: reading is not implemented")
}
