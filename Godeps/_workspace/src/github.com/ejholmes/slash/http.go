package slash

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

// Server adapts a Handler to be served over http.
type Server struct {
	Handler
}

// NewServer returns a new Server instance.
func NewServer(h Handler) *Server {
	return &Server{
		Handler: h,
	}
}

// ServeHTTP parses the Command from the incoming request then serves it using
// the Handler.
func (h *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	command, err := ParseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	responder := newResponder(command)
	resp, err := h.ServeCommand(context.Background(), responder, command)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newResponse(resp))
}

type response struct {
	ResponseType *string `json:"response_type,omitempty"`
	Text         string  `json:"text"`
}

func newResponse(resp Response) *response {
	r := &response{Text: resp.Text}
	if resp.InChannel {
		t := "in_channel"
		r.ResponseType = &t
	}
	return r
}
