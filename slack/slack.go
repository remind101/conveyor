// Package slack provides an slash Handler for adding the Conveyor push webhook
// on the GitHub repo.
package slack

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/ejholmes/slash"
	"github.com/google/go-github/github"
)

// NewHook returns a new github.Hook instance that represents the appropriate
// configuration for the Conveyor webhook.
func NewHook(url, secret string) *github.Hook {
	return &github.Hook{
		Events: []string{"push"},
		Active: github.Bool(true),
		Name:   github.String("web"),
		Config: map[string]interface{}{
			"url":          url,
			"content_type": "json",
			"secret":       secret,
		},
	}
}

// hooker is something that can add the conveyor webhook to a github repo.
type hooker interface {
	CreateHook(owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error)
	ListHooks(owner, repo string, opt *github.ListOptions) ([]github.Hook, *github.Response, error)
	EditHook(owner, repo string, id int, hook *github.Hook) (*github.Hook, *github.Response, error)
}

// WebhookHandler implements the slash.Handler interface for setting up the
// conveyor webhook.
type WebhookHandler struct {
	Hook *github.Hook
	hooker
}

// NewWebhookHandler initializes a new WebhookHandler instance.
func NewWebhookHandler(c *github.Client, hook *github.Hook) *WebhookHandler {
	return &WebhookHandler{
		Hook:   hook,
		hooker: c.Repositories,
	}
}

func (h *WebhookHandler) ServeCommand(ctx context.Context, command slash.Command) (reply string, err error) {
	params := slash.Params(ctx)
	owner, repo := params["owner"], params["repo"]

	var hook *github.Hook
	hook, err = h.existingHook(owner, repo)
	if err != nil {
		return
	}

	if hook != nil {
		if _, _, err = h.EditHook(owner, repo, *hook.ID, h.Hook); err != nil {
			return
		}

		reply = fmt.Sprintf("Updated webhook on %s/%s", owner, repo)
	} else {
		if _, _, err = h.CreateHook(owner, repo, h.Hook); err != nil {
			return
		}

		reply = fmt.Sprintf("Added webhook to %s/%s", owner, repo)
	}

	return
}

// existingHook returns an existing hook if it exists.
func (h *WebhookHandler) existingHook(owner, repo string) (*github.Hook, error) {
	hooks, _, err := h.ListHooks(owner, repo, nil)
	if err != nil {
		return nil, err
	}

	for _, hook := range hooks {
		if equalHooks(&hook, h.Hook) {
			return &hook, nil
		}
	}

	return nil, nil
}

func equalHooks(a, b *github.Hook) bool {
	if *a.Name == *b.Name {
		if *a.Name == "web" {
			return a.Config["url"].(string) == b.Config["url"].(string)
		}
	}

	return false
}
