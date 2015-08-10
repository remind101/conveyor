// +build s3

package conveyor

import (
	"io"
	"testing"
)

func TestS3Logger(t *testing.T) {
	f, _ := S3Logger("conveyor-data")
	w, _ := f(BuildOptions{
		Repository: "test",
		Branch:     "master",
		Sha:        "abcd",
	})

	io.WriteString(w, "Foobar")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
