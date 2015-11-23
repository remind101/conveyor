package conveyor

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/remind101/conveyor/builder"
)

// BuildLogs represents a service for reading and writing build logs.
type BuildLogs interface {
	Writer(builder.BuildOptions) (io.Writer, error)
	Reader(string) (io.Reader, error)
}

var DiscardLogs = &nullBuildLogs{}

// nullBuildLogs is a BuildLogs implementation that returns null readers and
// writers.
type nullBuildLogs struct{}

func (b *nullBuildLogs) Writer(opts builder.BuildOptions) (io.Writer, error) {
	return ioutil.Discard, nil
}

func (b *nullBuildLogs) Reader(id string) (io.Reader, error) {
	return strings.NewReader(""), nil
}
