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
// @termsOfService  http://example.com/terms/

// @contact.name   Your Name
// @contact.url    http://your-website.com
// @contact.email  you@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

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