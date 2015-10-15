package conveyor

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
)

func TestBuildQueue(t *testing.T) {
	q := &buildQueue{
		queue: make(chan buildRequest, 1),
	}

	background := context.Background()
	options := builder.BuildOptions{}
	err := q.Push(background, options)
	assert.NoError(t, err)

	ctx, got, err := q.Pop()
	assert.Equal(t, got, options)
	assert.Equal(t, ctx, background)
}
