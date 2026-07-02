package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"Mutesolo/internal/webapp"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address")
	staticDir := flag.String("static", "web", "static web directory")
	statePath := flag.String("state", webapp.DefaultStatePath(), "web state path")
	flag.Parse()

	server := webapp.NewServer(webapp.NewStore(*statePath), *staticDir)
	fmt.Printf("Mutesolo web console: http://%s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, server.Handler()))
}
