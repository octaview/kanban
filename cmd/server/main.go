package main

import (
	"log"

	_ "kanban/docs"
	"kanban/internal/config"
	"kanban/internal/server"
)

// @title           Kanban API
// @version         1.0
// @description     API for managing Kanban boards.

// @contact.name   octaview
// @contact.url    t.me/octaview
// @contact.email  octaviewes@gmail.com


// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @schemes http
func main() {
	cfg := config.Load()

	s, err := server.Init(cfg)
	if err != nil {
		log.Fatalf("❌ Server initialization failed: %v", err)
	}

	s.Run()
}