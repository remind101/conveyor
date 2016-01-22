package github

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/conveyor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const fakeUUID = "01234567-89ab-cdef-0123-456789abcdef"

func init() {
	newID = func() string { return fakeUUID }
}

func TestServer_Ping(t *testing.T) {
	s := NewServer(nil)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("X-GitHub-Event", "ping")

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestServer_Push(t *testing.T) {
	c := new(mockConveyor)
	s := newServer(c)

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

	c.On("Build", conveyor.BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "abcd",
	}).Return(&conveyor.Build{
		ID: fakeUUID,
	}, nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, resp.Body.String(), fakeUUID)
}

func TestServer_Push_Fork(t *testing.T) {
	c := new(mockConveyor)
	s := newServer(c)

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
	c := new(mockConveyor)
	s := newServer(c)

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

// mockConveyor is an implementation of the client interface.
type mockConveyor struct {
	mock.Mock
}

func (m *mockConveyor) Build(ctx context.Context, req conveyor.BuildRequest) (*conveyor.Build, error) {
	args := m.Called(req)
	return args.Get(0).(*conveyor.Build), args.Error(1)
}
