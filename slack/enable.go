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

// Enable implements the slash.Handler interface for enabling conveyor on a
// repo.
type Enable struct {
	Hook *github.Hook
	hooker
}

// NewEnable initializes a new Enable instance.
func NewEnable(c *github.Client, hook *github.Hook) *Enable {
	return &Enable{
		Hook:   hook,
		hooker: c.Repositories,
	}
}

func (h *Enable) ServeCommand(ctx context.Context, r slash.Responder, command slash.Command) (resp slash.Response, err error) {
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

		resp.Text = fmt.Sprintf("Updated webhook on %s/%s", owner, repo)
	} else {
		if _, _, err = h.CreateHook(owner, repo, h.Hook); err != nil {
			return
		}

		resp.Text = fmt.Sprintf("Added webhook to %s/%s", owner, repo)
	}

	return
}

// existingHook returns an existing hook if it exists.
func (h *Enable) existingHook(owner, repo string) (*github.Hook, error) {
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
