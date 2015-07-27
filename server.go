package conveyor

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/context"

	"github.com/codegangsta/negroni"
	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
)

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	Builder
	newLogger LogFactory
}

func NewServer(b Builder, newLogger LogFactory) http.Handler {
	if newLogger == nil {
		newLogger = StdoutLogger
	}

	s := &Server{Builder: b, newLogger: newLogger}

	r := hookshot.NewRouter()
	r.HandleFunc("ping", s.Ping)
	r.HandleFunc("push", s.Push)

	n := negroni.Classic()
	n.UseHandler(r)
	return n
}

func NewServerWithSecret(c *Conveyor, secret string) http.Handler {
	return hookshot.Authorize(NewServer(c, nil), secret)
}

func NewServerFromEnv() (http.Handler, error) {
	c, err := NewFromEnv()
	if err != nil {
		return nil, err
	}
	return NewServerWithSecret(c, os.Getenv("GITHUB_SECRET")), nil
}

// Ping is an http.HandlerFunc that will handle the `ping` event from GitHub.
func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ok\n")
}

// Push is an http.HandlerFunc that will handle the `push` event from GitHub.
func (s *Server) Push(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	var event events.Push

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	opts := BuildOptions{
		Repository: event.Repository.FullName,
		Branch:     strings.Replace(event.Ref, "refs/heads/", "", -1),
		Commit:     event.HeadCommit.ID,
	}

	log, err := s.newLogger(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opts.OutputStream = log

	if _, err := s.Build(ctx, opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
