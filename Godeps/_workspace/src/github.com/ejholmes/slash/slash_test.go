package slash

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testForm = `token=abcd&team_id=T012A0ABC&team_domain=acme&channel_id=D012A012A&channel_name=directmessage&user_id=U012A012A&user_name=ejholmes&command=%2Fdeploy&text=acme-inc+to+staging`

func TestCommandFromValues(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	assert.NoError(t, req.ParseForm())
	assert.Equal(t, CommandFromValues(req.Form), Command{
		Token:       "abcd",
		TeamID:      "T012A0ABC",
		TeamDomain:  "acme",
		ChannelID:   "D012A012A",
		ChannelName: "directmessage",
		UserID:      "U012A012A",
		UserName:    "ejholmes",
		Command:     "/deploy",
		Text:        "acme-inc to staging",
	})
}

func TestParseRequest(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got, err := ParseRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, got, Command{
		Token:       "abcd",
		TeamID:      "T012A0ABC",
		TeamDomain:  "acme",
		ChannelID:   "D012A012A",
		ChannelName: "directmessage",
		UserID:      "U012A012A",
		UserName:    "ejholmes",
		Command:     "/deploy",
		Text:        "acme-inc to staging",
	})
}

type mockHandler struct {
	mock.Mock
}

func (h *mockHandler) ServeCommand(ctx context.Context, command Command) (string, error) {
	args := h.Called(ctx, command)
	return args.String(0), args.Error(1)
}
