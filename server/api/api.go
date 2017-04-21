package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/gorilla/mux"
	"github.com/remind101/conveyor"
	schema "github.com/remind101/conveyor/client/conveyor"
	"github.com/remind101/pkg/stream"
	streamhttp "github.com/remind101/pkg/stream/http"
)

// client mocks out the interface from conveyor.Conveyor that we use.
type client interface {
	Logs(context.Context, string) (io.Reader, error)
	Build(context.Context, conveyor.BuildRequest) (*conveyor.Build, error)
	FindBuild(context.Context, string) (*conveyor.Build, error)
	FindArtifact(context.Context, string) (*conveyor.Artifact, error)
}

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	client

	// mux contains the routes.
	mux http.Handler
}

// NewServer returns a new Server instance
func NewServer(c *conveyor.Conveyor, auth func(http.Handler) http.Handler) *Server {
	return newServer(c, auth)
}

func newServer(c client, auth func(http.Handler) http.Handler) *Server {
	s := &Server{
		client: c,
	}

	authFunc := func(h http.HandlerFunc) http.Handler {
		return auth(http.HandlerFunc(h))
	}

	r := mux.NewRouter()
	// Builds
	r.Handle("/builds", authFunc(s.BuildCreate)).Methods("POST")
	r.Handle("/builds/{owner}/{repo}@{sha}", authFunc(s.BuildInfo)).Methods("GET")
	r.Handle("/builds/{id}", authFunc(s.BuildInfo)).Methods("GET")

	// Artifacts
	r.Handle("/artifacts/{owner}/{repo}@{sha}", authFunc(s.ArtifactInfo)).Methods("GET")
	r.Handle("/artifacts/{id}", authFunc(s.ArtifactInfo)).Methods("GET")

	// Logs
	r.HandleFunc("/logs/{id}", s.LogsStream).Methods("GET")

	s.mux = r
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// LogsStream is an http.HandlerFunc that will stream the logs for a build.
func (s *Server) LogsStream(rw http.ResponseWriter, req *http.Request) {
	ctx := context.TODO()

	vars := mux.Vars(req)

	// Get a handle to an io.Reader to stream the logs from.
	r, err := s.client.Logs(ctx, vars["id"])
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "text/plain")

	// Chrome won't show data if we don't set this. See
	// http://stackoverflow.com/questions/26164705/chrome-not-handling-chunked-responses-like-firefox-safari.
	rw.Header().Set("X-Content-Type-Options", "nosniff")

	// Needed for single page apps
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	w := streamhttp.StreamingResponseWriter(rw)
	defer close(stream.Heartbeat(w, time.Second*25)) // Send a null character every 25 seconds.

	// Copy the log stream to the client.
	_, err = io.Copy(w, r)
	if err != nil {
		fmt.Fprintf(w, "error: %v", err)
	}
}

// newBuild decorates a conveyor.Build as a schema.Build.
func newBuild(b *conveyor.Build) schema.Build {
	return schema.Build{
		ID:          b.ID,
		Repository:  b.Repository,
		Branch:      b.Branch,
		Sha:         b.Sha,
		State:       b.State.String(),
		CreatedAt:   b.CreatedAt,
		StartedAt:   b.StartedAt,
		CompletedAt: b.CompletedAt,
	}
}

// BuildCreate creates a Build and returns it.
func (s *Server) BuildCreate(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	var req schema.BuildCreateOpts
	if err := decode(r.Body, &req); err != nil {
		encodeErr(w, err)
		return
	}

	b, err := s.client.Build(ctx, conveyor.BuildRequest{
		Repository: req.Repository,
		Branch:     emptyString(req.Branch),
		Sha:        emptyString(req.Sha),
	})
	if err != nil {
		encodeErr(w, err)
		return
	}

	encode(w, newBuild(b))
}

// BuildInfo returns a Build.
func (s *Server) BuildInfo(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	ident := identity(mux.Vars(r))

	b, err := s.client.FindBuild(ctx, ident)
	if err != nil {
		encodeErr(w, err)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	encode(w, newBuild(b))
}

func newArtifact(a *conveyor.Artifact) schema.Artifact {
	artifact := schema.Artifact{
		ID:    a.ID,
		Image: a.Image,
	}
	artifact.Build.ID = a.BuildID
	return artifact
}

// ArtifactInfo returns an Artifact.
func (s *Server) ArtifactInfo(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	ident := identity(mux.Vars(r))

	a, err := s.client.FindArtifact(ctx, ident)
	if err != nil {
		encodeErr(w, err)
		return
	}

	encode(w, newArtifact(a))
	return
}

func identity(vars map[string]string) string {
	if id := vars["id"]; id != "" {
		return id
	}

	return fmt.Sprintf("%s/%s@%s", vars["owner"], vars["repo"], vars["sha"])
}

func encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func encodeErr(w http.ResponseWriter, e error) error {
	err := newError(e)

	switch err {
	case schema.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	return encode(w, err)
}

func newError(err error) *schema.Error {
	if err == sql.ErrNoRows {
		return schema.ErrNotFound
	}

	return &schema.Error{
		ID:      "internal_error",
		Message: err.Error(),
	}
}

// returns an empty string if the pointer is nil.
func emptyString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
