package conveyor

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServer_Logs(t *testing.T) {
	b := new(mockBuildLogs)
	s := NewServer(nil, b)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/logs/1234", nil)

	b.On("Reader", "1234").Return(strings.NewReader("Logs"), nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Logs", resp.Body.String())

	b.AssertExpectations(t)
}

func TestServer_Ping(t *testing.T) {
	s := NewServer(nil, nil)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("X-GitHub-Event", "ping")

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestServer_Push(t *testing.T) {
	q := new(mockBuildQueue)
	s := NewServer(q, nil)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`{
  "ref": "refs/heads/master",
  "head_commit": {
    "id": "abcd"
  },
  "repository": {
    "full_name": "remind101/acme-inc"
  }
}`))
	req.Header.Set("X-GitHub-Event", "push")

	q.On("Push", builder.BuildOptions{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	}).Return(nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestServer_Push_Fork(t *testing.T) {
	q := new(mockBuildQueue)
	s := NewServer(q, nil)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`{
  "ref": "refs/heads/master",
  "head_commit": {
    "id": "abcd"
  },
  "repository": {
    "full_name": "remind101/acme-inc",
    "fork": true
  }
}`))
	req.Header.Set("X-GitHub-Event", "push")

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestServer_Push_Deleted(t *testing.T) {
	q := new(mockBuildQueue)
	s := NewServer(q, nil)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`{
  "ref": "refs/heads/master",
  "deleted": true,
  "head_commit": {
    "id": "abcd"
  },
  "repository": {
    "full_name": "remind101/acme-inc"
  }
}`))
	req.Header.Set("X-GitHub-Event", "push")

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestNoCache(t *testing.T) {
	tests := []struct {
		in  string
		out bool
	}{
		// Use cache
		{"testing", false},

		// Don't use cache
		{"[docker nocache]", true},
		{"this is a commit [docker nocache]", true},
	}

	for _, tt := range tests {
		if got, want := noCache(tt.in), tt.out; got != want {
			t.Fatalf("noCache(%q) => %v; want %v", tt.in, got, want)
		}
	}
}

type mockBuildLogs struct {
	mock.Mock
}

func (b *mockBuildLogs) Writer(id string) (io.Writer, error) {
	args := b.Called(id)
	return args.Get(0).(io.Writer), args.Error(1)
}

func (b *mockBuildLogs) Reader(id string) (io.Reader, error) {
	args := b.Called(id)
	return args.Get(0).(io.Reader), args.Error(1)
}
