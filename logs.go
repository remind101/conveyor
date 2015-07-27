package conveyor

import (
	"io"
	"os"
)

// LogFactory is a function that can return a location to write logs to for a
// build.
type LogFactory func(BuildOptions) (io.Writer, error)

func StdoutLogger(opts BuildOptions) (io.Writer, error) {
	return os.Stdout, nil
}
