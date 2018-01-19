# honeybadger-go [![Build Status](https://travis-ci.org/honeybadger-io/honeybadger-go.svg?branch=master)](https://travis-ci.org/honeybadger-io/honeybadger-go)

Go (golang) support for the :zap: [Honeybadger error
notifier](https://www.honeybadger.io/). Receive instant notification of panics
and errors in your Go applications.

## Try it out

To deploy a sample Go application which uses this library to report errors to
Honeybadger.io:

[![Deploy](https://www.herokucdn.com/deploy/button.png)](https://heroku.com/deploy?template=https://github.com/honeybadger-io/crywolf-go)

Don't forget to destroy the Heroku app after you're done so that you aren't
charged for usage.

You can also [download the sample
application](https://github.com/honeybadger-io/crywolf-go) and run it locally.

## Installation

> **Note:** We recommend vendoring honeybadger-go in your application source until
we have a stable (v1.0.0) release. See [Versioning](#versioning) for more info.

To install, grab the package from GitHub:

```sh
go get github.com/honeybadger-io/honeybadger-go
```

Then add an import to your application code:

```go
import "github.com/honeybadger-io/honeybadger-go"
```

Finally, configure your API key:

```go
honeybadger.Configure(honeybadger.Configuration{APIKey: "your api key"})
```

You can also configure Honeybadger via environment variables. See
[Configuration](#configuration) for more information.

## Automatically reporting panics during a server request

To automatically report panics which happen during an HTTP request, wrap your
`http.Handler` function with `honeybadger.Handler`:

```go
log.Fatal(http.ListenAndServe(":8080", honeybadger.Handler(handler)))
```

Request data such as cookies and params will automatically be reported with
errors which happen inside `honeybadger.Handler`. Make sure you recover from
panics after honeybadger's Handler has been executed to ensure all panics are
reported.

## Automatically reporting other panics

To automatically report panics in your functions or methods, add
`defer honeybadger.Monitor()` to the beginning of the function or method you
wish to monitor. To report all unhandled panics which happen in your application
the following can be added to `main()`:

```go
func main() {
  defer honeybadger.Monitor()
  // application code...
}
```

Note that `honeybadger.Monitor()` will re-panic after it reports the error, so
make sure that it is only called once before recovering from the panic (or
allowing the process to crash).

## Manually reporting errors

To report an error manually, use `honeybadger.Notify`:

```go
if err != nil {
  honeybadger.Notify(err)
}
```

## Sending extra data to Honeybadger

To send extra context data to Honeybadger, use `honeybadger.SetContext`:

```go
honeybadger.SetContext(honeybadger.Context{
  "badgers": true,
  "user_id": 1,
})
```

You can also add local context using an optional second argument when calling
`honeybadger.Notify`:

```go
if err != nil {
  honeybadger.Notify(err, honeybadger.Context{"user_id": 2})
}
```

Local context keys override the keys set using `honeybadger.SetContext`.

## Creating a new client

In the same way that the log library provides a predefined "standard" logger,
honeybadger defines a standard client which may be accessed directly via
`honeybadger`. A new client may also be created by calling `honeybadger.New`:

```go
hb := honeybadger.New(honeybadger.Configuration{APIKey: "some other api key"})
hb.Notify("This error was reported by an alternate client.")
```

## Configuration

The following options are available through `honeybadger.Configuration`:

|  Name | Type | Default | Example | Environment variable |
| ----- | ---- | ------- | ------- | -------------------- |
| APIKey | `string` | `""` | `"badger01"` | `HONEYBADGER_API_KEY` |
| Root | `string` | The current working directory | `"/path/to/project"` | `HONEYBADGER_ROOT` |
| Env | `string` | `""` | `"production"` | `HONEYBADGER_ENV` |
| Hostname | `string` | The hostname of the current server. | `"badger01"` | `HONEYBADGER_HOSTNAME` |
| Endpoint | `string` | `"https://api.honeybadger.io"` | `"https://honeybadger.example.com/"` | `HONEYBADGER_ENDPOINT` |
| Timeout | `time.Duration` | 3 seconds | `10 * time.Second` | `HONEYBADGER_TIMEOUT` (nanoseconds) |
| Logger | `honeybadger.Logger` | Logs to stderr | `CustomLogger{}` | n/a |
| Backend | `honeybadger.Backend` | HTTP backend | `CustomBackend{}` | n/a |

## Versioning

We use [Semantic Versioning](http://semver.org/) to version releases of
honeybadger-go. Because there is no official method to specify version
dependencies in Go, we will do our best never to introduce a breaking change on
the master branch of this repo after reaching version 1. Until we reach version
1 there is a small chance that we may introduce a breaking change (changing the
signature of a function or method, for example), but we'll always tag a new
minor release and broadcast that we made the change.

If you're concerned about versioning, there are two options:

### Vendor your dependencies

If you're really concerned about changes to this library, then copy it into your
source control management system so that you can perform upgrades on your own
time.

### Use gopkg.in

Rather than importing directly from GitHub, [gopkg.in](http://gopkg.in/) allows
you to use their special URL format to transparently import a branch or tag from
GitHub. Because we tag each release, using gopkg.in can enable you to depend
explicitly on a certain version of this library. Importing from gopkg.in instead
of directly from GitHub is as easy as:

```go
import "gopkg.in/honeybadger-io/honeybadger-go.v0"
```

Check out the [gopkg.in](http://gopkg.in/) homepage for more information on how
to request versions.

## Changelog

See https://github.com/honeybadger-io/honeybadger-go/releases

## Contributing

If you're adding a new feature, please [submit an issue](https://github.com/honeybadger-io/honeybadger-go/issues/new) as a preliminary step; that way you can be (moderately) sure that your pull request will be accepted.

### To contribute your code:

1. Fork it.
2. Create a topic branch `git checkout -b my_branch`
3. Commit your changes `git commit -am "Boom"`
3. Push to your branch `git push origin my_branch`
4. Send a [pull request](https://github.com/honeybadger-io/honeybadger-go/pulls)

### License

This library is MIT licensed. See the [LICENSE](https://raw.github.com/honeybadger-io/honeybadger-go/master/LICENSE) file in this repository for details.
