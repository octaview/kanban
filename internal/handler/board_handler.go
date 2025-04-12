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
	boardRepo      *repository.BoardRepository
	boardShareRepo *repository.BoardShareRepository
}

func NewBoardHandler(boardRepo *repository.BoardRepository, boardShareRepo *repository.BoardShareRepository) *BoardHandler {
	return &BoardHandler{
		boardRepo:      boardRepo,
		boardShareRepo: boardShareRepo,
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

type UpdateBoardRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
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

func (h *BoardHandler) GetAll(c *gin.Context) {
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

	// Получаем доски, где пользователь - владелец
	ownedBoards, err := h.boardRepo.GetOwned(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve owned boards"})
		return
	}

	// Получаем доски, к которым пользователь имеет доступ
	sharedBoards, err := h.boardShareRepo.GetSharedBoards(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve shared boards"})
		return
	}

	// Объединяем результаты
	allBoards := append(ownedBoards, sharedBoards...)
	response := make([]BoardResponse, len(allBoards))
	
	for i, board := range allBoards {
		response[i] = BoardResponse{
			ID:          board.ID.String(),
			Title:       board.Title,
			Description: board.Description,
			OwnerID:     board.OwnerID.String(),
			CreatedAt:   board.CreatedAt.Format(http.TimeFormat),
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *BoardHandler) GetByID(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	authenticatedUserID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Parse board ID from URL
	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Get board from repository
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		return
	}

	// Проверяем, является ли пользователь владельцем доски или имеет к ней доступ
	if board.OwnerID != authenticatedUserID {
		hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), boardID, authenticatedUserID, model.RoleViewer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
			return
		}
		
		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to access this board"})
			return
		}
	}

	// Return board
	c.JSON(http.StatusOK, BoardResponse{
		ID:          board.ID.String(),
		Title:       board.Title,
		Description: board.Description,
		OwnerID:     board.OwnerID.String(),
		CreatedAt:   board.CreatedAt.Format(http.TimeFormat),
	})
}

func (h *BoardHandler) Update(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	authenticatedUserID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Parse board ID from URL
	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Get existing board from repository
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		return
	}

	// Проверяем права доступа на редактирование
	if board.OwnerID != authenticatedUserID {
		hasEditAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), boardID, authenticatedUserID, model.RoleEditor)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
			return
		}
		
		if !hasEditAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this board"})
			return
		}
	}

	// Parse request body
	var req UpdateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Update board fields if provided
	if req.Title != "" {
		board.Title = req.Title
	}
	if req.Description != "" {
		board.Description = req.Description
	}

	// Save the updated board
	if err := h.boardRepo.Update(c.Request.Context(), board); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update board"})
		return
	}

	// Return updated board
	c.JSON(http.StatusOK, BoardResponse{
		ID:          board.ID.String(),
		Title:       board.Title,
		Description: board.Description,
		OwnerID:     board.OwnerID.String(),
		CreatedAt:   board.CreatedAt.Format(http.TimeFormat),
	})
}