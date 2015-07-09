package conveyor

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/codegangsta/negroni"
	"github.com/ejholmes/hookshot"
)

// Server implements the http.Handler interface for serving build requests via
// GitHub webhooks.
type Server struct {
	builder
}

func NewServer(c *Conveyor) http.Handler {
	s := &Server{builder: newAsyncBuilder(c)}

	r := hookshot.NewRouter()
	r.HandleFunc("ping", s.Ping)
	r.HandleFunc("push", s.Push)

	n := negroni.Classic()
	n.UseHandler(r)
	return n
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

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ok\n")
}

func (s *Server) Push(w http.ResponseWriter, r *http.Request) {
	var f pushEvent

	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.Build(BuildOptions{
		Repository:   f.Repository.FullName,
		Branch:       strings.Replace(f.Ref, "refs/heads/", "", -1),
		Commit:       f.HeadCommit.ID,
		OutputStream: os.Stdout,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type pushEvent struct {
	Ref        string `json:"ref"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	HeadCommit struct {
		ID string `json:"id"`
	} `json:"head_commit"`
}

// builder represents something that can build a Docker image.
type builder interface {
	Build(BuildOptions) error
}

// asyncBuilder is an implementation of the builder interface that builds in a
// goroutine.
type asyncBuilder struct {
	builder
}

func newAsyncBuilder(b builder) *asyncBuilder {
	return &asyncBuilder{
		builder: b,
	}
}

func (b *asyncBuilder) Build(opts BuildOptions) error {
	go b.build(opts)
	return nil
}

func (b *asyncBuilder) build(opts BuildOptions) {
	if err := b.builder.Build(opts); err != nil {
		log.Printf("build err: %v", err)
	}
}
