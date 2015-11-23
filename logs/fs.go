package logs

import (
	"io"
	"os"
	"path/filepath"
)

// FSLogger is a Logger implementation that uses os.Open and os.Create.
type FSLogger struct {
	// A directory to store the logs in.
	Dir string
}

func (l *FSLogger) Create(name string) (io.Writer, error) {
	return os.Create(filepath.Join(l.Dir, name))
}

func (l *FSLogger) Open(name string) (io.Reader, error) {
	return os.Open(filepath.Join(l.Dir, name))
}
