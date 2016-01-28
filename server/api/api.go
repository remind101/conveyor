package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/gorilla/mux"
	"github.com/remind101/conveyor"
	"github.com/remind101/pkg/stream"
	streamhttp "github.com/remind101/pkg/stream/http"
)

// client mocks out the interface from conveyor.Conveyor that we use.
type client interface {
	Logs(context.Context, string) (io.Reader, error)
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
	r.HandleFunc("/logs/{id}", s.Logs).Methods("GET")

	s.mux = r
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Logs is an http.HandlerFunc that will stream the logs for a build.
func (s *Server) Logs(rw http.ResponseWriter, req *http.Request) {
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
	// TODO: Wrap w in a flush writer.
	_, err = io.Copy(w, r)
	if err != nil {
		fmt.Fprintf(w, "error: %v", err)
	}
}
