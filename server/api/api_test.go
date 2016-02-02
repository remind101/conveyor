package api

import (
	"io"
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

func nullAuth(h http.Handler) http.Handler {
	return h
}

func TestServer_Logs(t *testing.T) {
	c := new(mockConveyor)
	s := newServer(c, nullAuth)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/logs/1234", nil)

	c.On("Logs", "1234").Return(strings.NewReader("Logs"), nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Logs", resp.Body.String())

	c.AssertExpectations(t)
}

func TestServer_BuildCreate(t *testing.T) {
	c := new(mockConveyor)
	s := newServer(c, nullAuth)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/builds", strings.NewReader(`{
  "repository": "remind101/acme-inc",
  "branch": "master",
  "sha": "139759bd61e98faeec619c45b1060b4288952164"
}`))

	c.On("Build", conveyor.BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	}).Return(&conveyor.Build{
		ID:         fakeUUID,
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	}, nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "{\"branch\":\"master\",\"completed_at\":null,\"created_at\":\"0001-01-01T00:00:00Z\",\"id\":\"01234567-89ab-cdef-0123-456789abcdef\",\"repository\":\"remind101/acme-inc\",\"sha\":\"139759bd61e98faeec619c45b1060b4288952164\",\"started_at\":null,\"state\":\"pending\"}\n", resp.Body.String())

	c.AssertExpectations(t)
}

func TestServer_BuildInfo(t *testing.T) {
	c := new(mockConveyor)
	s := newServer(c, nullAuth)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/builds/01234567-89ab-cdef-0123-456789abcdef", nil)

	c.On("FindBuild", fakeUUID).Return(&conveyor.Build{
		ID:         fakeUUID,
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	}, nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "{\"branch\":\"master\",\"completed_at\":null,\"created_at\":\"0001-01-01T00:00:00Z\",\"id\":\"01234567-89ab-cdef-0123-456789abcdef\",\"repository\":\"remind101/acme-inc\",\"sha\":\"139759bd61e98faeec619c45b1060b4288952164\",\"started_at\":null,\"state\":\"pending\"}\n", resp.Body.String())

	c.AssertExpectations(t)
}

func TestServer_ArtifactInfo(t *testing.T) {
	c := new(mockConveyor)
	s := newServer(c, nullAuth)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/artifacts/01234567-89ab-cdef-0123-456789abcdef", nil)

	c.On("FindArtifact", fakeUUID).Return(&conveyor.Artifact{
		ID:      fakeUUID,
		Image:   "remind101/acme-inc:139759bd61e98faeec619c45b1060b4288952164",
		BuildID: fakeUUID,
	}, nil)

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "{\"build\":{\"id\":\"01234567-89ab-cdef-0123-456789abcdef\"},\"id\":\"01234567-89ab-cdef-0123-456789abcdef\",\"image\":\"remind101/acme-inc:139759bd61e98faeec619c45b1060b4288952164\"}\n", resp.Body.String())

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

func (m *mockConveyor) Build(ctx context.Context, req conveyor.BuildRequest) (*conveyor.Build, error) {
	args := m.Called(req)
	return args.Get(0).(*conveyor.Build), args.Error(1)
}

func (m *mockConveyor) FindBuild(ctx context.Context, buildIdentity string) (*conveyor.Build, error) {
	args := m.Called(buildIdentity)
	return args.Get(0).(*conveyor.Build), args.Error(1)
}

func (m *mockConveyor) FindArtifact(ctx context.Context, artifactIdentity string) (*conveyor.Artifact, error) {
	args := m.Called(artifactIdentity)
	return args.Get(0).(*conveyor.Artifact), args.Error(1)
}
