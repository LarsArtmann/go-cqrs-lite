package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const usage = `taskmanager: event-sourced task API demo on go-cqrs-lite

Starts an HTTP server backed by an in-process SQLite event store,
projections, and a seeded demo task. Ctrl-C shuts down gracefully.

Routes:
  GET    /health               liveness probe
  GET    /api/tasks            list tasks
  POST   /api/tasks            create a task ({"title": "..."})
  GET    /api/tasks/{id}       task detail (also /assign, /complete, ...)
  PATCH  /api/tasks/{id}       update task state
  DELETE /api/tasks/{id}       tombstone the task stream
  GET    /api/stats            task statistics
  GET    /events               SSE stream of live view updates

Flags:
`

func main() {
	cfg := DefaultConfig()
	if p := os.Getenv("DATABASE_PATH"); p != "" {
		cfg.DatabasePath = p
	}

	help := flag.Bool("help", false, "print this usage and exit")
	flag.StringVar(&cfg.HTTPAddr, "addr", cfg.HTTPAddr, "HTTP listen address")
	flag.StringVar(
		&cfg.DatabasePath,
		"db",
		cfg.DatabasePath,
		`SQLite database path (":memory:" keeps everything in RAM)`,
	)
	flag.Parse()

	if *help {
		fmt.Print(usage)
		flag.PrintDefaults()

		return
	}

	if err := Run(cfg); err != nil {
		log.Fatal(err)
	}
}
