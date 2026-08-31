// Command server runs the taskboard HTTP API.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"localtask/internal/config"
	"localtask/internal/httpapi"
	"localtask/internal/registry"
	"localtask/internal/watch"
)

func main() {
	configPath := flag.String("config", "taskboard.config.yaml", "path to the taskboard config file")
	port := flag.String("port", "8080", "port to listen on")
	staticDir := flag.String("static", "", "path to a built frontend (web/dist) to serve; if empty, only /api/ is served (the mode used in local dev, where Vite serves the frontend separately)")
	host := flag.String("host", "127.0.0.1", "host to bind to. Leave at the default for direct (non-Docker) use — it is what keeps the app off the LAN. Inside a container, override to 0.0.0.0: Docker's own port-publish flag (e.g. \"127.0.0.1:8080:8080\") is what enforces localhost-only from the host, since it forwards into the container's own network namespace rather than its loopback")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("no config file at %s — starting with zero projects configured (see compose.yaml for the expected format); the app will run, but the project list will be empty until you add one via the UI, or edit the config and restart", *configPath)
			cfg = config.New(*configPath)
		} else {
			log.Fatalf("loading config: %v", err)
		}
	}

	projects, errs := registry.Load(cfg)
	for _, err := range errs {
		log.Printf("skipping %v", err)
	}
	if len(projects) == 0 {
		log.Printf("no projects loaded — the app will run, but the project list will be empty")
	}
	for name := range projects {
		log.Printf("loaded project %q", name)
	}

	watcher, err := watch.Start(projects, nil)
	if err != nil {
		log.Printf("filesystem watching disabled: %v", err)
	}

	reg := registry.NewRegistry(cfg, projects, watcher)
	srv := httpapi.New(reg, *staticDir)

	addr := fmt.Sprintf("%s:%s", *host, *port)
	log.Printf("listening on http://%s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
