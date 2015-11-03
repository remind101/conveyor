package slash

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServer(t *testing.T) {
	h := new(mockHandler)
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	h.On("ServeCommand",
		context.Background(),
		mock.AnythingOfType("Command"),
	).Return("ok", nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestServer_Err(t *testing.T) {
	h := new(mockHandler)
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	errBoom := errors.New("boom")
	h.On("ServeCommand",
		context.Background(),
		mock.AnythingOfType("Command"),
	).Return("", errBoom)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}
