package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServer_Logs(t *testing.T) {
	c := new(mockConveyor)
	s := newServer(c)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/logs/1234", nil)

	c.On("Logs", "1234").Return(strings.NewReader("Logs"), nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Logs", resp.Body.String())

	c.AssertExpectations(t)
}

// mockConveyor is an implementation of the client interface.
type mockConveyor struct {
	mock.Mock
}

func (m *mockConveyor) Logs(ctx context.Context, buildID string) (io.Reader, error) {
	args := m.Called(buildID)
	return args.Get(0).(io.Reader), args.Error(1)
}
