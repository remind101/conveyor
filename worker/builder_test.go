package worker

import (
	"bytes"
	"testing"

	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestBuilder_Build(t *testing.T) {
	b := new(mockBuilder)
	bb := &Builder{
		builder: b,
	}

	w := new(bytes.Buffer)
	options := builder.BuildOptions{}

	b.On("Build", w, options).Return("", nil)

	_, err := bb.Build(context.Background(), w, options)
	assert.NoError(t, err)
}
