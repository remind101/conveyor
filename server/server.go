package server

import (
	"net/http"

	"github.com/ejholmes/hookshot"
	"github.com/gorilla/mux"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/server/api"
	"github.com/remind101/conveyor/server/github"
)

type Config struct {
	APIAuth func(http.Handler) http.Handler

	// Shared secret between GitHub and Conveyor.
	GitHubSecret string
}

func NewServer(c *conveyor.Conveyor, config Config) http.Handler {
	r := mux.NewRouter()

	// Github webhooks
	r.MatcherFunc(githubWebhook).Handler(
		hookshot.Authorize(github.NewServer(c), config.GitHubSecret),
	)

	// API
	r.NotFoundHandler = api.NewServer(c, config.APIAuth)

	return r
}

// githubWebhook is a MatcherFunc that matches requests that have an
// `X-GitHub-Event` header present.
func githubWebhook(r *http.Request, _ *mux.RouteMatch) bool {
	h := r.Header[http.CanonicalHeaderKey("X-GitHub-Event")]
	return len(h) > 0
}
