package slashtest_test

import (
	"fmt"
	"net/http/httptest"
	"time"

	"golang.org/x/net/context"

	"github.com/ejholmes/slash"
	"github.com/ejholmes/slash/slashtest"
)

func ExampleServer() {
	// A slash.Handler that will handle our slash commands.
	h := slash.NewServer(slash.HandlerFunc(func(ctx context.Context, r slash.Responder, c slash.Command) error {
		return r.Respond(slash.Reply("Hey"))
	}))

	// Responses from the above handler will be posted here.
	responses := slashtest.NewServer()
	defer responses.Close()

	req, _ := slashtest.NewRequest("POST", "/", responses.NewCommand())
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	select {
	case resp := <-responses.Responses:
		fmt.Println(resp.Text)
		// Output: Hey
	case <-time.After(time.Second):
		panic("timeout")
	}
}
