package slash

import (
	"errors"
	"regexp"

	"golang.org/x/net/context"
)

var (
	// ErrNoHandler is returned by Mux ServeCommand if a Handler isn't found
	// for the route.
	ErrNoHandler = errors.New("slash: no handler")

	// ErrUnauthorized is returned when the provided token in the request
	// does not match the expected secret.
	ErrUnauthorized = errors.New("slash: invalid token")
)

// Handler represents something that handles a slash command.
type Handler interface {
	// ServeCommand runs the command. The handler should return a string
	// that will be used as the reply to send back to the user, or an error.
	// If an error is returned, then the string value is what will be sent
	// to the user.
	ServeCommand(context.Context, Command) (reply string, err error)
}

// HandlerFunc is a function that implements the Handler interface.
type HandlerFunc func(context.Context, Command) (string, error)

func (fn HandlerFunc) ServeCommand(ctx context.Context, command Command) (string, error) {
	return fn(ctx, command)
}

// Matcher is something that can check if a Command matches a Route.
type Matcher interface {
	Match(Command) (map[string]string, bool)
}

// MatcherFunc is a function that implements Matcher.
type MatcherFunc func(Command) (map[string]string, bool)

func (fn MatcherFunc) Match(command Command) (map[string]string, bool) {
	return fn(command)
}

// MatchCommand returns a Matcher that checks that the command strings match.
func MatchCommand(cmd string) Matcher {
	return MatcherFunc(func(command Command) (map[string]string, bool) {
		return make(map[string]string), command.Command == cmd
	})
}

// MatchTextRegexp returns a Matcher that checks that the command text matches a
// regular expression.
func MatchTextRegexp(r *regexp.Regexp) Matcher {
	return MatcherFunc(func(command Command) (map[string]string, bool) {
		params := make(map[string]string)
		matches := r.FindStringSubmatch(command.Text)
		if len(matches) == 0 {
			return params, false
		}

		for i, m := range matches {
			k := r.SubexpNames()[i]
			if k != "" {
				params[k] = m
			}
		}

		return params, true
	})
}

// Route wraps a Handler with a Matcher.
type Route struct {
	Handler
	Matcher
}

// NewRoute returns a new Route instance.
func NewRoute(handler Handler) *Route {
	return &Route{
		Handler: handler,
	}
}

// Mux is a Handler implementation that routes commands to Handlers.
type Mux struct {
	routes []*Route
}

// NewMux returns a new Mux instance.
func NewMux() *Mux {
	return &Mux{}
}

// Handle adds a Handler to handle the given command.
//
// Example
//
//	m.Handle("/deploy", "token", DeployHandler)
func (m *Mux) Command(command, token string, handler Handler) *Route {
	return m.Match(MatchCommand(command), ValidateToken(handler, token))
}

// MatchText adds a route that matches when the text of the command matches the
// given regular expression. If the route matches and is called, slash.Matches
// will return the capture groups.
func (m *Mux) MatchText(re *regexp.Regexp, handler Handler) *Route {
	return m.Match(MatchTextRegexp(re), handler)
}

// Match adds a new route that uses the given Matcher to match.
func (m *Mux) Match(matcher Matcher, handler Handler) *Route {
	r := NewRoute(handler)
	r.Matcher = matcher
	return m.addRoute(r)
}

func (m *Mux) addRoute(r *Route) *Route {
	m.routes = append(m.routes, r)
	return r
}

// Handler returns the Handler that can handle the given slash command. If no
// handler matches, nil is returned.
func (m *Mux) Handler(command Command) (Handler, map[string]string) {
	for _, r := range m.routes {
		if params, ok := r.Match(command); ok {
			return r.Handler, params
		}
	}
	return nil, nil
}

// ServeCommand attempts to find a Handler to serve the Command. If no handler
// is found, an error is returned.
func (m *Mux) ServeCommand(ctx context.Context, command Command) (string, error) {
	h, params := m.Handler(command)
	if h == nil {
		return "", ErrNoHandler
	}
	return h.ServeCommand(WithParams(ctx, params), command)
}

// ValidateToken returns a new Handler that verifies that the token in the
// request matches the given token.
func ValidateToken(h Handler, token string) Handler {
	return HandlerFunc(func(ctx context.Context, command Command) (string, error) {
		if command.Token != token {
			return "", ErrUnauthorized
		}
		return h.ServeCommand(ctx, command)
	})
}
