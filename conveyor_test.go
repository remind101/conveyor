package conveyor

import (
	"errors"
	"testing"

	"golang.org/x/net/context"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const fakeUUID = "01234567-89ab-cdef-0123-456789abcdef"

const databaseURL = "postgres://localhost/conveyor?sslmode=disable"

func init() {
	newID = func() string { return fakeUUID }
}

func TestConveyor_Build(t *testing.T) {
	q := new(mockBuildQueue)
	c := newConveyor(t)
	c.BuildQueue = q

	q.On("Push", builder.BuildOptions{
		ID:         "<build_id>",
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	}).Once().Return(nil)

	b, err := c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotEqual(t, "", b.ID)

	b, err = c.FindBuild(context.Background(), b.ID)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotNil(t, b.ID)
	assert.Equal(t, StatePending, b.State)
	assert.Equal(t, "remind101/acme-inc", b.Repository)
	assert.Equal(t, "master", b.Branch)
	assert.Equal(t, "139759bd61e98faeec619c45b1060b4288952164", b.Sha)
}

func TestConveyor_Build_Duplicate(t *testing.T) {
	q := new(mockBuildQueue)
	c := newConveyor(t)
	c.BuildQueue = q

	q.On("Push", builder.BuildOptions{
		ID:         "<build_id>",
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	}).Once().Return(nil)

	b, err := c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)

	_, err = c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.Equal(t, ErrDuplicateBuild, err)

	err = c.BuildStarted(context.Background(), b.ID)
	assert.NoError(t, err)

	_, err = c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.Equal(t, ErrDuplicateBuild, err)
}

func TestConveyor_BuildStarted(t *testing.T) {
	c := newConveyor(t)

	b, err := c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)

	err = c.BuildStarted(context.Background(), b.ID)
	assert.NoError(t, err)

	b, err = c.FindBuild(context.Background(), b.ID)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotNil(t, b.StartedAt)
	assert.Equal(t, StateBuilding, b.State)
}

func TestConveyor_BuildComplete(t *testing.T) {
	c := newConveyor(t)

	b, err := c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)

	image := "remind101/acme-inc:139759bd61e98faeec619c45b1060b4288952164"
	err = c.BuildComplete(context.Background(), b.ID, image)
	assert.NoError(t, err)

	b, err = c.FindBuild(context.Background(), b.ID)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotNil(t, b.CompletedAt)
	assert.Equal(t, StateSucceeded, b.State)
}

func TestConveyor_BuildFailed(t *testing.T) {
	c := newConveyor(t)

	b, err := c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)

	err = c.BuildFailed(context.Background(), b.ID, errors.New("Docker error"))
	assert.NoError(t, err)

	b, err = c.FindBuild(context.Background(), b.ID)
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotNil(t, b.CompletedAt)
	assert.Equal(t, StateFailed, b.State)
}

func TestConveyor_FindArtifact(t *testing.T) {
	q := new(mockBuildQueue)
	c := newConveyor(t)
	c.BuildQueue = q

	q.On("Push", builder.BuildOptions{
		ID:         "<build_id>",
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	}).Once().Return(nil)

	b, err := c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)

	image := "remind101/acme-inc:139759bd61e98faeec619c45b1060b4288952164"
	err = c.BuildComplete(context.Background(), b.ID, image)
	assert.NoError(t, err)

	// Find by repo@sha
	a, err := c.FindArtifact(context.Background(), "remind101/acme-inc@139759bd61e98faeec619c45b1060b4288952164")
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, image, a.Image)

	// Find by id
	a, err = c.FindArtifact(context.Background(), a.ID)
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, image, a.Image)
}

// This tests the case where we have a previous successful build for a sha that
// resulted in an artifact, but we re-triggered the build. We want to return the
// artifacts from the previous successful build until the new build is
// successful.
func TestConveyor_FindArtifact_PendingBuild(t *testing.T) {
	q := new(mockBuildQueue)
	c := newConveyor(t)
	c.BuildQueue = q

	q.On("Push", builder.BuildOptions{
		ID:         "<build_id>",
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	}).Twice().Return(nil)

	b, err := c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)

	image := "remind101/acme-inc:139759bd61e98faeec619c45b1060b4288952164"
	err = c.BuildComplete(context.Background(), b.ID, image)
	assert.NoError(t, err)

	successfulBuild := b

	// Start a new build
	b, err = c.Build(context.Background(), BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)

	a, err := c.FindArtifact(context.Background(), "remind101/acme-inc@139759bd61e98faeec619c45b1060b4288952164")
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, image, a.Image)
	assert.Equal(t, successfulBuild.ID, a.BuildID)

	// Mark the new build as complete. New artifact.
	err = c.BuildComplete(context.Background(), b.ID, image)
	assert.NoError(t, err)

	newBuild := b
	a, err = c.FindArtifact(context.Background(), "remind101/acme-inc@139759bd61e98faeec619c45b1060b4288952164")
	assert.NoError(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, image, a.Image)
	assert.Equal(t, newBuild.ID, a.BuildID)
}

func newConveyor(t testing.TB) *Conveyor {
	db := sqlx.MustConnect("postgres", databaseURL)
	if err := Reset(db); err != nil {
		t.Fatal(err)
	}

	c := New(db)
	c.BuildQueue = NewBuildQueue(100)

	return c
}

type mockBuildQueue struct {
	mock.Mock
}

func (m *mockBuildQueue) Push(ctx context.Context, options builder.BuildOptions) error {
	options.ID = "<build_id>" // ID will be a uuid that we can't mock.
	args := m.Called(options)
	return args.Error(0)
}

func (m *mockBuildQueue) Subscribe(chan BuildContext) error {
	return nil
}
