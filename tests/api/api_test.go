package api_test

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"github.com/jmoiron/sqlx"
	core "github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/client/conveyor"
	"github.com/remind101/conveyor/logs"
	"github.com/remind101/conveyor/server"
	"github.com/remind101/conveyor/worker"
	"github.com/stretchr/testify/assert"
)

const databaseURL = "postgres://localhost/conveyor_api?sslmode=disable"

func TestBuild(t *testing.T) {
	c := newClient(t)
	defer c.Close(t)

	buf := new(bytes.Buffer)
	a, err := c.Build(buf, conveyor.BuildCreateOpts{
		Repository: "remind101/acme-inc",
		Branch:     conveyor.String("master"),
		Sha:        conveyor.String("139759bd61e98faeec619c45b1060b4288952164"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "remind101/acme-inc:1234", a.Image)
	assert.NotEqual(t, "", buf.String())
}

type Client struct {
	*conveyor.Service
	s *httptest.Server
	c *Conveyor
}

func newClient(t testing.TB) *Client {
	c := newConveyor(t)
	s := httptest.NewServer(server.NewServer(c.Conveyor, server.Config{}))

	cl := conveyor.NewService(conveyor.DefaultClient)
	cl.URL = s.URL

	return &Client{
		Service: cl,
		s:       s,
		c:       c,
	}
}

func (c *Client) Close(t testing.TB) {
	c.s.Close()
	c.c.Close(t)
}

// Wraps a Conveyor instance and a set of workers together.
type Conveyor struct {
	*core.Conveyor
	worker *worker.Worker
}

func newConveyor(t testing.TB) *Conveyor {
	db := sqlx.MustConnect("postgres", databaseURL)
	if err := core.Reset(db); err != nil {
		t.Fatal(err)
	}

	c := core.New(db)
	c.BuildQueue = core.NewBuildQueue(100)
	c.Logger = logs.Discard

	ch := make(chan core.BuildContext)
	c.BuildQueue.Subscribe(ch)

	w := worker.New(c, worker.Options{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
			io.WriteString(w, "Pulling base image\n")
			return "remind101/acme-inc:1234", nil
		}),
		BuildRequests: ch,
	})

	go w.Start()

	return &Conveyor{
		Conveyor: c,
		worker:   w,
	}
}

func (c *Conveyor) Close(t testing.TB) {
	if err := c.worker.Shutdown(); err != nil {
		t.Fatal(err)
	}
}
