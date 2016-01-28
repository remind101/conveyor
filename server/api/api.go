package api

import (
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
func NewServer(c *conveyor.Conveyor) *Server {
	return newServer(c)
}

func newServer(c client) *Server {
	s := &Server{
		client: c,
	}

	r := mux.NewRouter()
	// Builds
	r.HandleFunc("/builds", s.BuildCreate).Methods("POST")
	r.HandleFunc("/builds/{build_id}", s.BuildInfo).Methods("GET")

	// Artifacts
	r.HandleFunc("/artifacts/{artifact_id_or_image}", s.ArtifactInfo).Methods("GET")

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
		Branch:     req.Branch,
		Sha:        req.Sha,
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

	ident := mux.Vars(r)["build_id"]

	b, err := s.client.FindBuild(ctx, ident)
	if err != nil {
		encodeErr(w, err)
		return
	}

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

	ident := mux.Vars(r)["artifact_id_or_image"]

	a, err := s.client.FindArtifact(ctx, ident)
	if err != nil {
		encodeErr(w, err)
		return
	}

	encode(w, newArtifact(a))
	return
}

func encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func encodeErr(w io.Writer, err error) error {
	return encode(w, newError(err))
}

type Error schema.Error

func (e *Error) Error() string {
	return e.Message
}

func newError(err error) Error {
	return Error{
		ID:      "internal_error",
		Message: err.Error(),
	}
}
