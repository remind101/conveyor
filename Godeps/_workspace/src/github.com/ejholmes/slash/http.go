package slash

import (
	"io"
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

	text, err := h.ServeCommand(context.Background(), command)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	io.WriteString(w, text)
}
