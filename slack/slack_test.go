package slack

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/ejholmes/slash"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	githubToken = "accesstoken"
	slackToken  = "slacktoken"
)

func TestWebhookHandler(t *testing.T) {
	hook := &github.Hook{}
	g := new(mockHooker)
	h := &WebhookHandler{
		Hook:   hook,
		hooker: g,
	}

	g.On("ListHooks", "remind101", "acme-inc").Return([]github.Hook{}, nil)
	g.On("CreateHook", "remind101", "acme-inc", hook).Return(nil)

	ctx := slash.WithParams(context.Background(), map[string]string{
		"owner": "remind101",
		"repo":  "acme-inc",
	})
	reply, err := h.ServeCommand(ctx, slash.Command{
		Token:   slackToken,
		Command: "/conveyor",
		Text:    "setup remind101/acme-inc",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Added webhook to remind101/acme-inc", reply)

	g.AssertExpectations(t)
}

func TestWebhookHandler_Exists(t *testing.T) {
	hook := &github.Hook{
		Name: github.String("web"),
		Config: map[string]interface{}{
			"url": "http://www.google.com",
		},
	}
	g := new(mockHooker)
	h := &WebhookHandler{
		Hook:   hook,
		hooker: g,
	}

	g.On("ListHooks", "remind101", "acme-inc").Return([]github.Hook{
		{
			ID:   github.Int(1),
			Name: github.String("web"),
			Config: map[string]interface{}{
				"url": "http://www.google.com",
			},
		},
	}, nil)
	g.On("EditHook", "remind101", "acme-inc", 1, hook).Return(nil)

	ctx := slash.WithParams(context.Background(), map[string]string{
		"owner": "remind101",
		"repo":  "acme-inc",
	})
	reply, err := h.ServeCommand(ctx, slash.Command{
		Token:   slackToken,
		Command: "/conveyor",
		Text:    "setup remind101/acme-inc",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Updated webhook on remind101/acme-inc", reply)

	g.AssertExpectations(t)
}

type mockHooker struct {
	mock.Mock
}

func (h *mockHooker) CreateHook(owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error) {
	args := h.Called(owner, repo, hook)
	return nil, nil, args.Error(0)
}

func (h *mockHooker) ListHooks(owner, repo string, opt *github.ListOptions) ([]github.Hook, *github.Response, error) {
	args := h.Called(owner, repo)
	return args.Get(0).([]github.Hook), nil, args.Error(1)
}

func (h *mockHooker) EditHook(owner, repo string, id int, hook *github.Hook) (*github.Hook, *github.Response, error) {
	args := h.Called(owner, repo, id, hook)
	return nil, nil, args.Error(0)
}
