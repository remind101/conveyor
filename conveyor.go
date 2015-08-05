package conveyor

import "io"

const (
	// Context is used for the commit status context.
	Context = "container/docker"

	// DefaultBuilderImage is the docker image used to build docker images.
	DefaultBuilderImage = "remind101/conveyor-builder"
)

type BuildOptions struct {
	// Repository is the repo to build.
	Repository string
	// Sha is the git commit to build.
	Sha string
	// Branch is the name of the branch that this build relates to.
	Branch string
	// An io.Writer where output will be written to.
	OutputStream io.Writer
}
