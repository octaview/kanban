package handler

import (
	"net/http"

	"kanban/internal/model"
	"kanban/internal/repository"
	"kanban/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const MaxBoardsPerUser = 5

type BoardHandler struct {
	boardRepo *repository.BoardRepository
}

func NewBoardHandler(boardRepo *repository.BoardRepository) *BoardHandler {
	return &BoardHandler{
		boardRepo: boardRepo,
	}
}

type CreateBoardRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type BoardResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
	CreatedAt   string `json:"created_at"`
}

// Create creates a new board for the authenticated user
func (h *BoardHandler) Create(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	ownerID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Check if user already has 5 boards
	count, err := h.boardRepo.CountOwned(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board count"})
		return
	}

	if count >= MaxBoardsPerUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "Maximum number of boards reached (5)"})
		return
	}

	// Parse request body
	var req CreateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Create new board
	board := &model.Board{
		Title:       req.Title,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	if err := h.boardRepo.Create(c.Request.Context(), board); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create board"})
		return
	}

	// Return created board
	c.JSON(http.StatusCreated, BoardResponse{
		ID:          board.ID.String(),
		Title:       board.Title,
		Description: board.Description,
		OwnerID:     board.OwnerID.String(),
		CreatedAt:   board.CreatedAt.Format(http.TimeFormat),
	})
}