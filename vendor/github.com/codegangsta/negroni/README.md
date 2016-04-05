# Negroni [![GoDoc](https://godoc.org/github.com/codegangsta/negroni?status.png)](http://godoc.org/github.com/codegangsta/negroni)

Negroni is an idiomatic approach to web middleware in Go. It is tiny, non-intrusive, and encourages use of `net/http` Handlers.

If you like the idea of [Martini](http://github.com/go-martini/martini), but you think it contains too much magic, then Negroni is a great fit.

## Getting Started

After installing Go and setting up your [GOPATH](http://golang.org/doc/code.html#GOPATH), create your first `.go` file. We'll call it `server.go`.

~~~ go
package main

import (
  "github.com/codegangsta/negroni"
  "net/http"
  "fmt"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
    fmt.Fprintf(w, "Welcome to the home page!")
  })
  
  n := negroni.Classic()
  n.UseHandler(mux)
  n.Run(":3000")
}
~~~

Then install the Negroni package (**go 1.1** and greater is required):
~~~
go get github.com/codegangsta/negroni
~~~

Then run your server:
~~~
go run server.go
~~~

You will now have a Go net/http webserver running on `localhost:3000`.

## Need Help?
If you have a question or feature request, [go ask the mailing list](https://groups.google.com/forum/#!forum/negroni-users). The GitHub issues for Negroni will be used exclusively for bug reports and pull requests.

## Is Negroni a Framework?
Negroni is **not** a framework. It is a library that is designed to work directly with net/http.

## Routing?
Negroni is BYOR (Bring your own Router). The Go community already has a number of great http routers available, Negroni tries to play well with all of them by fully supporting `net/http`. For instance, integrating with [Gorilla Mux](http://github.com/gorilla/mux) looks like so:

~~~ go
router := mux.NewRouter()
router.HandleFunc("/", HomeHandler)

n := negroni.New()
n.Use(Middleware1)
n.Use(Middleware2)
n.Use(Middleware3)
// router goes last
n.UseHandler(router)

n.Run(":3000")
~~~

## `negroni.Classic()`
`negroni.Classic()` provides some default middleware that is useful for most applications:

* `negroni.Recovery` - Panic Recovery Middleware.
* `negroni.Logging` - Request/Response Logging Middleware.
* `negroni.Static` - Static File serving under the "public" directory.

This makes it really easy to get started with some useful features from Negroni.

## Handlers
Negroni provides a bidirectional middleware flow. This is done through the `negroni.Handler` interface:

~~~ go
type Handler interface {
  ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}
~~~

If a middlware hasn't already written to the ResponseWriter, it should call the next `http.HandlerFunc` in the chain to yield to the next middleware handler. This can be used for great good:

~~~ go
func MyMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
  // do some stuff before
  next(rw, r)
  // do some stuff after
}
~~~

And you can map it to the handler chain with the `Use` function:

~~~ go
n := negroni.New()
n.Use(negroni.HandlerFunc(MyMiddleware))
~~~

You can also map plain ole `http.Handler`'s:

~~~ go
n := negroni.New()

mux := http.NewServeMux()
// map your routes

n.UseHandler(mux)

n.Run(":3000")
~~~

## `Run()`
Negroni has a convenience function called `Run`. `Run` takes an addr string identical to [http.ListenAndServe](http://golang.org/pkg/net/http#ListenAndServe).

~~~ go
n := negroni.Classic()
// ...
log.Fatal(http.ListenAndServe(":8080", n))
~~~

## Third Party Middleware

Here is a current list of Negroni compatible middlware. Feel free to put up a PR linking your middleware if you have built one:


| Middleware | Author | Description |
| -----------|--------|-------------|
| [Graceful](https://github.com/stretchr/graceful) | [Tyler Bunnell](https://github.com/tylerb) | Graceful HTTP Shutdown |
| [negroni-secure](https://github.com/unrolled/negroni-secure) | [Cory Jacobsen](https://github.com/unrolled) | Middleware that implements a few quick security wins. |


## Live code reload?
[gin](https://github.com/codegangsta/gin) and [fresh](https://github.com/pilu/fresh) both live reload negroni apps.

## About

Negroni is obsessively designed by none other than the [Code Gangsta](http://codegangsta.io/)
