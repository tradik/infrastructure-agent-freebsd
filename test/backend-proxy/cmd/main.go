package main

import (
	"net/http"
	"os"
	"time"

	"github.com/newrelic/infrastructure-agent/pkg/log"
	"github.com/newrelic/infrastructure-agent/test/backend-proxy/pkg/server"
)

var (
	// internal
	mlog = log.WithComponent("test.backend")
)

func main() {
	r := server.GetRouter()
	addr := "127.0.0.1:8080"
	srv := &http.Server{
		Handler: r.GetHandler(),
		Addr:    addr,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	mlog.Infof("Starting server at address %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		os.Exit(1)
	}
}
