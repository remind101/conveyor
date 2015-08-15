Builders are the backbone of Conveyor. They're what takes a git commit, and turns it into a Docker image.

## Builders

The following builder implementations are provided:

* [Docker](./docker): This is a Builder implementation that builds Docker images inside Docker. It also tags the resulting image with the branch and git commit sha before pushing it to the docker registry.

Adding your own builder is easy. Just implement the following interface:

```go
// Builder represents something that can build a Docker image.
type Builder interface {
	// Builder should build an image and write output to w.Writer. In general,
	// it's expected that the image will be pushed to some location where it
	// can be pulled by clients.
	//
	// Implementers should take note and handle the ctx.Done() case in the
	// event that the build should timeout or get canceled by the user.
	//
	// The value of image should be the location to pull the immutable
	// image. For example, if the image is built and generates a sha256
	// digest, the value for image may look like:
	//
	//	remind101/acme-inc@sha256:6b558cade79544da908c349ba0e5b63d
	//
	// Or possibly a tag:
	//
	//	remind101/acme-inc:<git sha>
	Build(ctx context.Context, w io.Writer, opts BuildOptions) (image string, err error)
}
```
