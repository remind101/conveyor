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
)

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	Builder
	LogFactory LogFactory

	// mux contains the routes.
	mux http.Handler
}

// NewServer returns a new Server instance
func NewServer(b Builder) *Server {
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

	opts := BuildOptions{
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

	opts.OutputStream = log

	if _, err := s.Build(ctx, opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) newLogger(opts BuildOptions) (io.Writer, error) {
	if s.LogFactory == nil {
		return StdoutLogger(opts)
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
