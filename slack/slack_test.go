package slack

import (
	"testing"
	"text/template"

	"github.com/ejholmes/slash"
	"github.com/remind101/conveyor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

const fakeUUID = "01234567-89ab-cdef-0123-456789abcdef"

func TestBuild(t *testing.T) {
	c := new(mockConveyor)
	b := &Slack{
		client:      c,
		URLTemplate: template.Must(template.New("url").Parse("http://conveyor/logs/{{.ID}}")),
	}

	ctx := slash.WithParams(context.Background(), map[string]string{
		"owner":  "remind101",
		"repo":   "acme-inc",
		"branch": "master",
	})

	c.On("Build", conveyor.BuildRequest{
		Repository: "remind101/acme-inc",
		Branch:     "master",
	}).Return(&conveyor.Build{
		ID: fakeUUID,
	}, nil)

	rec := &fakeResponder{responses: make(chan slash.Response, 2)}
	err := b.Build(ctx, rec, slash.Command{})
	resp := <-rec.responses
	assert.Equal(t, "One moment...", resp.Text)
	assert.NoError(t, err)

	resp = <-rec.responses
	assert.Equal(t, "Building remind101/acme-inc@master: http://conveyor/logs/01234567-89ab-cdef-0123-456789abcdef", resp.Text)
}

// mockConveyor is an implementation of the client interface.
type mockConveyor struct {
	mock.Mock
}

func (m *mockConveyor) Build(ctx context.Context, req conveyor.BuildRequest) (*conveyor.Build, error) {
	args := m.Called(req)
	return args.Get(0).(*conveyor.Build), args.Error(1)
}

func (m *mockConveyor) EnableRepo(ctx context.Context, repo string) error {
	args := m.Called(repo)
	return args.Error(0)
}

type fakeResponder struct {
	responses chan slash.Response
}

func (r *fakeResponder) Respond(resp slash.Response) error {
	r.responses <- resp
	return nil
}
