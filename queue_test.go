package conveyor

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
)

func TestBuildQueue(t *testing.T) {
	q := &buildQueue{
		queue: make(chan BuildRequest, 1),
	}

	background := context.Background()
	options := builder.BuildOptions{}
	err := q.Push(background, options)
	assert.NoError(t, err)

	req := <-q.Subscribe()
	assert.Equal(t, req.BuildOptions, options)
	assert.Equal(t, req.Ctx, background)
}
