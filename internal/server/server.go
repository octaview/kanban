package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kanban/internal/config"
	"kanban/internal/handler"
	"kanban/internal/middleware"
	"kanban/internal/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Server struct {
	Engine *gin.Engine
	DB     *gorm.DB
	Config *config.Config
}

func Init(cfg *config.Config) (*Server, error) {
	// Setup GORM
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("❌ failed to connect to DB: %w", err)
	}
	log.Println("✅ Connected to database")

	// Setup Gin
	r := gin.Default()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	boardRepo := repository.NewBoardRepository(db)
	boardShareRepo := repository.NewBoardShareRepository(db)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userRepo)
	boardHandler := handler.NewBoardHandler(boardRepo, boardShareRepo)
	boardShareHandler := handler.NewBoardShareHandler(boardRepo, userRepo, boardShareRepo)

	// Public routes
	r.POST("/register", userHandler.Register)
	r.POST("/login", userHandler.Login)

	// Protected routes - require authentication
	authorized := r.Group("/")
	authorized.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
	{
		// Board routes
		authorized.POST("/boards", boardHandler.Create)
		authorized.GET("/boards", boardHandler.GetAll)
		authorized.GET("/boards/:id", boardHandler.GetByID)
		authorized.PUT("/boards/:id", boardHandler.Update)
		
		// Board sharing routes
		authorized.POST("/boards/:id/share", boardShareHandler.ShareBoard)
		authorized.DELETE("/boards/:id/share/:user_id", boardShareHandler.RemoveShare)
		authorized.GET("/boards/:id/share", boardShareHandler.GetBoardShares)
		authorized.GET("/shared-boards", boardShareHandler.GetSharedBoards)
	}
	return &Server{
		Engine: r,
		DB:     db,
		Config: cfg,
	}, nil
}

func (s *Server) Run() {
	srv := &http.Server{
		Addr:    ":" + s.Config.ServerPort,
		Handler: s.Engine,
	}

	go func() {
		log.Printf("🚀 Server running on port %s\n", s.Config.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to listen: %s\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %s", err)
	}

	log.Println("✅ Server exited properly")
}