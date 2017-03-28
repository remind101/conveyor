// package hb2 is a Go package for sending errors to Honeybadger
// using the official client library
package hb2

import (
	"net/http"
	"strings"

	"github.com/honeybadger-io/honeybadger-go"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

// Headers that won't be sent to honeybadger.
var IgnoredHeaders = map[string]struct{}{
	"Authorization": struct{}{},
}

type Config struct {
	ApiKey      string
	Environment string
	Endpoint    string
}

type hbReporter struct {
	client *honeybadger.Client
}

// NewReporter returns a new Reporter instance.
func NewReporter(cfg Config) *hbReporter {
	hbCfg := honeybadger.Configuration{}
	hbCfg.APIKey = cfg.ApiKey
	hbCfg.Env = cfg.Environment
	hbCfg.Endpoint = cfg.Endpoint

	return &hbReporter{honeybadger.New(hbCfg)}
}

// Report reports the error to honeybadger.
func (r *hbReporter) Report(ctx context.Context, err error) error {
	extras := []interface{}{}

	if e, ok := err.(*reporter.Error); ok {
		extras = append(extras, getContextData(e))
		if r := e.Request; r != nil {
			extras = append(extras, honeybadger.Params(r.Form), getRequestData(r), *r.URL)
		}
		err = e.Err
	}

	_, clientErr := r.client.Notify(err, extras...)
	return clientErr
}

func getRequestData(r *http.Request) honeybadger.CGIData {
	cgiData := honeybadger.CGIData{}
	replacer := strings.NewReplacer("-", "_")

	for header, values := range r.Header {
		if _, ok := IgnoredHeaders[header]; ok {
			continue
		}
		key := "HTTP_" + replacer.Replace(strings.ToUpper(header))
		cgiData[key] = strings.Join(values, ",")
	}

	cgiData["REQUEST_METHOD"] = r.Method
	return cgiData
}

func getContextData(err *reporter.Error) honeybadger.Context {
	ctx := honeybadger.Context{}
	for key, value := range err.Context {
		ctx[key] = value
	}
	return ctx
}
