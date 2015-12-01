package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServer_Logs(t *testing.T) {
	l := new(mockLogger)
	s := NewServer(l)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/logs/1234", nil)

	l.On("Open", "1234").Return(strings.NewReader("Logs"), nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Logs", resp.Body.String())

	l.AssertExpectations(t)
}

type mockLogger struct {
	mock.Mock
}

func (b *mockLogger) Create(name string) (io.Writer, error) {
	args := b.Called(name)
	return args.Get(0).(io.Writer), args.Error(1)
}

func (b *mockLogger) Open(name string) (io.Reader, error) {
	args := b.Called(name)
	return args.Get(0).(io.Reader), args.Error(1)
}
