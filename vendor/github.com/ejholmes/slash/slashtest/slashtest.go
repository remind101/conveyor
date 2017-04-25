// Package slashtest contains helpers for testing slash commands.
package slashtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/ejholmes/slash"
)

// ResponseRecorder is a slash.Responder implementation for testing purposes. It
// records the responses in a channel that can then be received on to make
// assertions. It also attempts to mimick the behavior of Slack in that it will
// return an error if you try to send more than 5 responses.
type ResponseRecorder struct {
	Responses <-chan slash.Response

	// internal channel to send on.
	ch chan slash.Response
}

// NewRecorder returns a new ResponseRecorder with the Responses channel set to
// a buffered channel allowing 5 responses.
func NewRecorder() *ResponseRecorder {
	ch := make(chan slash.Response, slash.MaximumDelayedResponses)
	return &ResponseRecorder{
		Responses: ch,
		ch:        ch,
	}
}

// Respond sends the response on the Responses channel. If the channel is
// blocked, it returns an error.
func (r *ResponseRecorder) Respond(resp slash.Response) error {
	return r.add(resp)
}

// ServeHTTP makes the ResponseRecorder implement the http.Handler method so it
// can be used in combination with httptest.Server to record responses posted to
// Slack.
func (r *ResponseRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var resp slash.Response
	if err := json.NewDecoder(req.Body).Decode(&resp); err != nil {
		panic(err)
	}

	if err := r.add(resp); err != nil {
		panic(err)
	}
}

func (r *ResponseRecorder) add(resp slash.Response) error {
	select {
	case r.ch <- resp:
		return nil
	default:
		return fmt.Errorf("you can send a maximum of %d delayed responses", cap(r.ch))
	}
}

// Server provides an http server that can handle slash.Responses posted to the
// response_url. It's simply a combination of an httptest.Server and a
// ResponseRecorder. You should dispose of it when you're done by calling the
// Close method.
type Server struct {
	*httptest.Server
	*ResponseRecorder
}

// NewServer returns an httptest.Server that uses a ResponseRecorder as the
// http.Handler
func NewServer() *Server {
	r := NewRecorder()
	s := httptest.NewServer(r)
	return &Server{
		Server:           s,
		ResponseRecorder: r,
	}
}

// NewCommand returns a new slash.Command with the ResponseURL attribute set so
// that handlers will post to this server.
func (s *Server) NewCommand() slash.Command {
	u, err := url.Parse(s.URL)
	if err != nil {
		// Should never happen
		panic(err)
	}
	return slash.Command{ResponseURL: u}
}

// NewRequest is a small helper that returns an http.Request suitable to POST to
// a slash.Server.
func NewRequest(method, path string, cmd slash.Command) (*http.Request, error) {
	v := slash.ValuesFromCommand(cmd)
	req, err := http.NewRequest(method, path, strings.NewReader(v.Encode()))
	if err != nil {
		return req, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}
