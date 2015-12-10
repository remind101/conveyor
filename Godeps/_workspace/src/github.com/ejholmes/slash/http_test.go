package slash

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
)

func TestServer_Reply(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, r Responder, command Command) (Response, error) {
		return Reply("ok"), nil
	})
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, `{"text":"ok"}`+"\n", resp.Body.String())
}

func TestServer_Say(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, r Responder, command Command) (Response, error) {
		return Say("ok"), nil
	})
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, `{"response_type":"in_channel","text":"ok"}`+"\n", resp.Body.String())
}

func TestServer_Err(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, r Responder, command Command) (Response, error) {
		return NoResponse, errors.New("boom")
	})
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}
