// Package slack provides an slash Handler for adding the Conveyor push webhook
// on the GitHub repo.
package slack

import (
	"golang.org/x/net/context"

	"github.com/ejholmes/slash"
	"github.com/remind101/conveyor"
)

// client mocks out the interface from conveyor.Conveyor that we use.
type client interface {
	Build(context.Context, conveyor.BuildRequest) (*conveyor.Build, error)
}

// replyHandler returns a slash.Handler that just replies to the user with the
// text.
func replyHandler(text string) slash.Handler {
	return slash.HandlerFunc(func(ctx context.Context, r slash.Responder, c slash.Command) (slash.Response, error) {
		return slash.Reply(text), nil
	})
}
