package main

import (
	"log"

	"kanban/internal/config"
	"kanban/internal/server"
)

func main() {
	cfg := config.Load()

	s, err := server.Init(cfg)
	if err != nil {
		log.Fatalf("❌ Server initialization failed: %v", err)
	}

	s.Run()
}