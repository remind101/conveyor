package conveyor

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context"

	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
	"github.com/gorilla/mux"
	"github.com/remind101/conveyor/builder"
)

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	Queue BuildQueue

	BuildLogs BuildLogs

	// mux contains the routes.
	mux http.Handler
}

// NewServer returns a new Server instance
func NewServer(q BuildQueue, b BuildLogs) *Server {
	s := &Server{Queue: q, BuildLogs: b}

	g := hookshot.NewRouter()
	g.HandleFunc("ping", s.Ping)
	g.HandleFunc("push", s.Push)

	r := mux.NewRouter()
	r.HandleFunc("/logs/{id}", s.Logs).Methods("GET")
	r.NotFoundHandler = g

	s.mux = r
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Logs is an http.HandlerFunc that will stream the logs for a build.
func (s *Server) Logs(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	// Get a handle to an io.Reader to stream the logs from.
	r, err := s.BuildLogs.Reader(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Copy the log stream to the client.
	io.Copy(w, r)
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

	// Enqueue the build
	if err := s.Queue.Push(ctx, opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// http://rubular.com/r/y8oJAY9eAS
var noCacheRegexp = regexp.MustCompile(`\[docker nocache\]`)

// noCache returns whether the docker layer cache should be used for this build
// or not.
func noCache(message string) bool {
	return noCacheRegexp.MatchString(message)
}
