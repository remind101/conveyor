package api_test

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"github.com/jmoiron/sqlx"
	"github.com/remind101/conveyor"
	"github.com/remind101/conveyor/builder"
	client "github.com/remind101/conveyor/client/conveyor"
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
	a, err := c.Build(buf, client.BuildCreateOpts{
		Repository: "remind101/acme-inc",
		Branch:     "master",
		Sha:        "139759bd61e98faeec619c45b1060b4288952164",
	})
	assert.NoError(t, err)
	assert.Equal(t, "remind101/acme-inc:1234", a.Image)
}

type Client struct {
	*client.Service
	s *httptest.Server
	c *Conveyor
}

func newClient(t testing.TB) *Client {
	c := newConveyor(t)
	s := httptest.NewServer(server.NewServer(c.Conveyor, server.Config{}))

	cl := client.NewService(client.DefaultClient)
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
	*conveyor.Conveyor
	worker *worker.Worker
}

func newConveyor(t testing.TB) *Conveyor {
	db := sqlx.MustConnect("postgres", databaseURL)
	if err := conveyor.Reset(db); err != nil {
		t.Fatal(err)
	}

	c := conveyor.New(db)
	c.BuildQueue = conveyor.NewBuildQueue(100)
	c.Logger = logs.Discard

	ch := make(chan conveyor.BuildContext)
	c.BuildQueue.Subscribe(ch)

	w := worker.New(c, worker.Options{
		Builder: builder.BuilderFunc(func(ctx context.Context, w io.Writer, options builder.BuildOptions) (string, error) {
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
