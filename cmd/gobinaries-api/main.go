package main

import (
	"context"
	"net/http"

	"github.com/apex/log"

	"github.com/google/go-github/v28/github"
	"github.com/tj/go/env"
	"golang.org/x/oauth2"

	"github.com/skrashevich/gobinaries/resolver"
	"github.com/skrashevich/gobinaries/server"
	"github.com/skrashevich/gobinaries/storage"
)

// main
func main() {

	// context
	ctx := context.Background()

	// github client
	gh := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: env.Get("GITHUB_TOKEN"),
		},
	)

	// server
	addr := ":" + env.GetDefault("PORT", "3000")
	s := &server.Server{
		Static: "static",
		URL:    env.GetDefault("URL", "http://127.0.0.1"+addr),
		Resolver: &resolver.GitHub{
			Client: github.NewClient(oauth2.NewClient(ctx, gh)),
		},
		Storage: &storage.Local{

			Prefix: "production",
		},
	}

	// listen
	log.WithField("addr", addr).Info("starting server")
	err := http.ListenAndServe(addr, s)
	if err != nil {
		log.Fatalf("error: %s", err)
	}
}

// Flusher interface.
type Flusher interface {
	Flush() error
}

// flusher returns an HTTP handler which flushes after each request.
func flusher(h http.Handler, f Flusher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)

		err := f.Flush()
		if err != nil {
			log.WithError(err).Error("error flushing logs")
			return
		}
	})
}
