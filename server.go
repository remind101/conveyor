package conveyor

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/ejholmes/hookshot"
	"github.com/google/go-github/github"
)

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	conveyor *Conveyor
}

func NewServer(c *Conveyor) *Server {
	return &Server{
		conveyor: c,
	}
}

func NewServerWithSecret(c *Conveyor, secret string) http.Handler {
	return hookshot.Authorize(NewServer(c), secret)
}

func NewServerFromEnv() (http.Handler, error) {
	c, err := NewFromEnv()
	if err != nil {
		return nil, err
	}
	return NewServerWithSecret(c, os.Getenv("GITHUB_SECRET")), nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var f github.PushEvent

	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.conveyor.Build(BuildOptions{
		Repository: *f.Repo.FullName,
		Branch:     strings.Replace(*f.Ref, "refs/heads/", "", -1),
		Commit:     *f.Head,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
