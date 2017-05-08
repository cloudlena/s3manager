# Adapters

Adapters is a collection of useful HTTP middleware or "Adapters". They follow the Adapter Pattern described by Mat Ryer in his blog post [Writing middleware in #golang and how Go makes it so much fun.](https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81)

Adapters can be chained in any way and will be executed in the order they are specified.

## Usage

```go
package main

import (
        "fmt"
        "log"
        "net/http"

        "github.com/mastertinner/adapters"
        "github.com/mastertinner/adapters/logging"
)

// IndexHandler says what it loves
func IndexHandler() http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
        })
}

func main() {
        logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
        http.Handle("/", adapters.Adapt(IndexHandler(), logging.Handler(logger)))
        log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Adapters

This package contains the following adapters:

* Logging: Logs the request and the time it took to serve it
* OAuth2: Checks if a request is authenticated through [OAuth 2](https://oauth.net/2/) using [Redis](https://redis.io/) as a cache
