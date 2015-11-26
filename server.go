package conveyor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
	"github.com/gorilla/mux"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/logs"
	"github.com/remind101/pkg/stream"
	streamhttp "github.com/remind101/pkg/stream/http"
)

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	Queue BuildQueue

	Logger logs.Logger

	// mux contains the routes.
	mux http.Handler
}

// ServerConfig is provided when initializing a new Server instance
type ServerConfig struct {
	// Secret is the shared secret for authenticating the GitHub webhooks.
	Secret string

	// Queue is the BuildQueue to use to enqueue new builds.
	Queue BuildQueue

	// Logger is the logger to use to stream logs to clients.
	Logger logs.Logger
}

// NewServer returns a new Server instance
func NewServer(config ServerConfig) *Server {
	s := &Server{Queue: config.Queue, Logger: config.Logger}

	g := hookshot.NewRouter()
	g.HandleFunc("ping", s.Ping)
	g.HandleFunc("push", s.Push)

	r := mux.NewRouter()
	r.HandleFunc("/logs/{id}", s.Logs).Methods("GET")
	r.MatcherFunc(githubWebhook).Handler(
		hookshot.Authorize(g, config.Secret),
	)

	s.mux = r
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Logs is an http.HandlerFunc that will stream the logs for a build.
func (s *Server) Logs(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	// Get a handle to an io.Reader to stream the logs from.
	r, err := s.Logger.Open(vars["id"])
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "text/plain")

	// Chrome won't show data if we don't set this. See
	// http://stackoverflow.com/questions/26164705/chrome-not-handling-chunked-responses-like-firefox-safari.
	rw.Header().Set("X-Content-Type-Options", "nosniff")

	w := streamhttp.StreamingResponseWriter(rw)
	defer close(stream.Heartbeat(w, time.Second*25)) // Send a null character every 25 seconds.

	// Copy the log stream to the client.
	// TODO: Wrap w in a flush writer.
	_, err = io.Copy(w, r)
	if err != nil {
		fmt.Fprintf(w, "error: %v", err)
	}
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

	id := newID()
	opts := builder.BuildOptions{
		ID:         id,
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

	io.WriteString(w, id)
}

// http://rubular.com/r/y8oJAY9eAS
var noCacheRegexp = regexp.MustCompile(`\[docker nocache\]`)

// noCache returns whether the docker layer cache should be used for this build
// or not.
func noCache(message string) bool {
	return noCacheRegexp.MatchString(message)
}

// githubWebhook is a MatcherFunc that matches requests that have an
// `X-GitHub-Event` header present.
func githubWebhook(r *http.Request, _ *mux.RouteMatch) bool {
	h := r.Header[http.CanonicalHeaderKey("X-GitHub-Event")]
	return len(h) > 0
}
