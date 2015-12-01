package cloudwatch

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	b := bytes.NewReader([]byte{'H', 'e', 'l', 'l', 'o'})
	r := &reader{Reader: b}

	buf := make([]byte, 100)
	n, err := r.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
}

func TestReader_Closed(t *testing.T) {
	b := bytes.NewReader([]byte{'H', 'e', 'l', 'l', 'o', '\x03'})
	r := &reader{Reader: b}

	buf := make([]byte, 100)
	n, err := r.Read(buf)
	assert.Equal(t, 6, n)
	assert.Equal(t, io.EOF, err)
}

func TestReader_NoData(t *testing.T) {
	b := new(emptyReader)
	r := &reader{Reader: b}

	buf := make([]byte, 100)
	n, err := r.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

type emptyReader struct{}

func (r *emptyReader) Read(b []byte) (int, error) {
	return 0, nil
}
