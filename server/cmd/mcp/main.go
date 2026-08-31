// Command taskboard-mcp runs the taskboard MCP server over stdio, so a
// coding agent can launch it locally alongside a project (see spec §17).
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"localtask/internal/config"
	"localtask/internal/mcpserver"
	"localtask/internal/registry"
	"localtask/internal/watch"
)

func main() {
	// log's default output is os.Stderr, which is required here: stdout is
	// reserved for the MCP JSON-RPC stream over StdioTransport.
	configPath := flag.String("config", "taskboard.config.yaml", "path to the taskboard config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("no config file at %s — starting with zero projects configured", *configPath)
			cfg = &config.Config{}
		} else {
			log.Fatalf("loading config: %v", err)
		}
	}

	projects, errs := registry.Load(cfg)
	for _, err := range errs {
		log.Printf("skipping %v", err)
	}
	if len(projects) == 0 {
		log.Printf("no projects loaded — list_projects will return an empty list until the config is fixed and the server restarted")
	}
	for name := range projects {
		log.Printf("loaded project %q", name)
	}

	if _, err := watch.Start(projects, nil); err != nil {
		log.Printf("filesystem watching disabled: %v", err)
	}

	srv := mcpserver.New(projects)
	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
