package slack

import (
	"testing"
	"text/template"

	"github.com/ejholmes/slash"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

const fakeUUID = "01234567-89ab-cdef-0123-456789abcdef"

func init() {
	newID = func() string { return fakeUUID }
}

func TestBuild(t *testing.T) {
	q := new(mockBuildQueue)
	r := new(mockBranchResolver)
	b := &Build{
		Queue:          q,
		branchResolver: r,
		urlTmpl:        template.Must(template.New("url").Parse("http://conveyor/logs/{{.ID}}")),
	}

	ctx := slash.WithParams(context.Background(), map[string]string{
		"owner":  "remind101",
		"repo":   "acme-inc",
		"branch": "master",
	})

	r.On("resolveBranch", "remind101", "acme-inc", "master").Return("sha", nil)
	q.On("Push", ctx, builder.BuildOptions{
		ID:         fakeUUID,
		Repository: "remind101/acme-inc",
		Sha:        "sha",
		Branch:     "master",
	}).Return(nil)

	rec := &fakeResponder{responses: make(chan slash.Response, 1)}
	resp, err := b.ServeCommand(ctx, rec, slash.Command{})
	assert.Equal(t, "One moment...", resp.Text)
	assert.NoError(t, err)

	resp = <-rec.responses
	assert.Equal(t, "Building remind101/acme-inc:master http://conveyor/logs/01234567-89ab-cdef-0123-456789abcdef", resp.Text)
}

func TestBuildRegexp(t *testing.T) {
	tests := []struct {
		in  string
		out map[string]string
	}{
		{`build remind101/acme-inc@topic-branch`, map[string]string{"owner": "remind101", "repo": "acme-inc", "branch": "topic-branch"}},
		{`build remind101/acme-inc:topic-branch`, map[string]string{"owner": "remind101", "repo": "acme-inc", "branch": "topic-branch"}},
		{`build remind101/acme-inc#topic-branch`, map[string]string{"owner": "remind101", "repo": "acme-inc", "branch": "topic-branch"}},
	}

	for _, tt := range tests {
		m := slash.MatchTextRegexp(BuildRegexp)
		out, ok := m.Match(slash.Command{Text: tt.in})
		assert.True(t, ok)
		assert.Equal(t, tt.out, out)
	}
}

type mockBuildQueue struct {
	mock.Mock
}

func (m *mockBuildQueue) Push(ctx context.Context, options builder.BuildOptions) error {
	args := m.Called(ctx, options)
	return args.Error(0)
}

func (m *mockBuildQueue) Subscribe(chan conveyor.BuildContext) error {
	return nil
}

type mockBranchResolver struct {
	mock.Mock
}

func (m *mockBranchResolver) resolveBranch(owner, repo, branch string) (string, error) {
	args := m.Called(owner, repo, branch)
	return args.String(0), args.Error(1)
}

type fakeResponder struct {
	responses chan slash.Response
}

func (r *fakeResponder) Respond(resp slash.Response) error {
	r.responses <- resp
	return nil
}
