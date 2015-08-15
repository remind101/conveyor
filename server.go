package conveyor

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context"

	"github.com/codegangsta/negroni"
	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/conveyor/builder"
)

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	builder.Builder
	LogFactory builder.LogFactory

	// mux contains the routes.
	mux http.Handler
}

// NewServer returns a new Server instance
func NewServer(b *Conveyor) *Server {
	s := &Server{Builder: b}

	r := hookshot.NewRouter()
	r.HandleFunc("ping", s.Ping)
	r.HandleFunc("push", s.Push)

	n := negroni.Classic()
	n.UseHandler(r)

	s.mux = n
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
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

	// Don't build forks.
	if event.Repository.Fork {
		io.WriteString(w, "Not building fork")
		return
	}

	// Don't build deleted branches.
	if event.Deleted {
		io.WriteString(w, "Not building deleted branch")
		return
	}

	opts := builder.BuildOptions{
		Repository: event.Repository.FullName,
		Branch:     strings.Replace(event.Ref, "refs/heads/", "", -1),
		Sha:        event.HeadCommit.ID,
		NoCache:    noCache(event.HeadCommit.Message),
	}

	log, err := s.newLogger(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := s.Build(ctx, log, opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) newLogger(opts builder.BuildOptions) (builder.Logger, error) {
	if s.LogFactory == nil {
		return builder.StdoutLogger(opts)
	}

	return s.LogFactory(opts)
}

// http://rubular.com/r/y8oJAY9eAS
var noCacheRegexp = regexp.MustCompile(`\[docker nocache\]`)

// noCache returns whether the docker layer cache should be used for this build
// or not.
func noCache(message string) bool {
	return noCacheRegexp.MatchString(message)
}
