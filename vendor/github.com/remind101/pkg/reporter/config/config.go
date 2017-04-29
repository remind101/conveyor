package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2"
	"github.com/remind101/pkg/reporter/rollbar"
)

// Returns a MultiReporter from URL strings such as:
// "hb://api.honeybadger.io/?key=hbkey&environment=hbenv" or
// "rollbar://api.rollbar.com/?key=rollbarkey&environment=rollbarenv"
func NewReporterFromUrls(urls []string) reporter.Reporter {
	rep := reporter.MultiReporter{}
	for _, url := range urls {
		rep = append(rep, newReporterFromUrl(url))
	}
	return rep
}

func newReporterFromUrl(url string) reporter.Reporter {
	u := urlParse(url)
	switch u.Scheme {
	case "hb":
		q := u.Query()
		return hb2.NewReporter(hb2.Config{
			ApiKey:      q.Get("key"),
			Environment: q.Get("environment"),
		})
	case "rollbar":
		q := u.Query()
		rollbar.ConfigureReporter(q.Get("key"), q.Get("environment"))
		return rollbar.Reporter
	default:
		must(fmt.Errorf("unrecognized reporter url scheme: %s", url))
		return nil
	}
}

func urlParse(uri string) *url.URL {
	u, err := url.Parse(uri)
	if err != nil {
		must(err)
	}
	return u
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
