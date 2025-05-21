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

// Create godoc
// @Summary Create a new board
// @Description Create a new Kanban board for the authenticated user
// @Tags Boards
// @Accept json
// @Produce json
// @Param request body CreateBoardRequest true "Board creation details"
// @Success 201 {object} BoardResponse "Board created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Not authenticated"
// @Failure 403 {object} map[string]string "Maximum number of boards reached"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Router /boards [post]
func (h *BoardHandler) Create(c *gin.Context) {
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

	count, err := h.boardRepo.CountOwned(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board count"})
		return
	}

	if count >= MaxBoardsPerUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "Maximum number of boards reached (5)"})
		return
	}

	var req CreateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	board := &model.Board{
		Title:       req.Title,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	if err := h.boardRepo.Create(c.Request.Context(), board); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create board"})
		return
	}

	c.JSON(http.StatusCreated, BoardResponse{
		ID:          board.ID.String(),
		Title:       board.Title,
		Description: board.Description,
		OwnerID:     board.OwnerID.String(),
		CreatedAt:   board.CreatedAt.Format(http.TimeFormat),
	})
}

// GetAll godoc
// @Summary Get all accessible boards
// @Description Get all boards that the authenticated user owns or has access to
// @Tags Boards
// @Produce json
// @Success 200 {array} BoardResponse "List of boards"
// @Failure 401 {object} map[string]string "Not authenticated"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Router /boards [get]
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

	ownedBoards, err := h.boardRepo.GetOwned(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve owned boards"})
		return
	}

	sharedBoards, err := h.boardShareRepo.GetSharedBoards(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve shared boards"})
		return
	}

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

// GetByID godoc
// @Summary Get a board by ID
// @Description Get a specific board by its ID if the authenticated user has access
// @Tags Boards
// @Produce json
// @Param id path string true "Board ID"
// @Success 200 {object} BoardResponse "Board details"
// @Failure 400 {object} map[string]string "Invalid board ID format"
// @Failure 401 {object} map[string]string "Not authenticated"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Board not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Router /boards/{id} [get]
func (h *BoardHandler) GetByID(c *gin.Context) {
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

	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		return
	}

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

	c.JSON(http.StatusOK, BoardResponse{
		ID:          board.ID.String(),
		Title:       board.Title,
		Description: board.Description,
		OwnerID:     board.OwnerID.String(),
		CreatedAt:   board.CreatedAt.Format(http.TimeFormat),
	})
}

// Update godoc
// @Summary Update a board
// @Description Update a board's title and/or description if the authenticated user has permission
// @Tags Boards
// @Accept json
// @Produce json
// @Param id path string true "Board ID"
// @Param request body UpdateBoardRequest true "Board update details"
// @Success 200 {object} BoardResponse "Updated board details"
// @Failure 400 {object} map[string]string "Invalid request or board ID format"
// @Failure 401 {object} map[string]string "Not authenticated"
// @Failure 403 {object} map[string]string "Permission denied"
// @Failure 404 {object} map[string]string "Board not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Router /boards/{id} [put]
func (h *BoardHandler) Update(c *gin.Context) {
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

	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		return
	}

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

	var req UpdateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Title != "" {
		board.Title = req.Title
	}
	if req.Description != "" {
		board.Description = req.Description
	}

	if err := h.boardRepo.Update(c.Request.Context(), board); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update board"})
		return
	}

	c.JSON(http.StatusOK, BoardResponse{
		ID:          board.ID.String(),
		Title:       board.Title,
		Description: board.Description,
		OwnerID:     board.OwnerID.String(),
		CreatedAt:   board.CreatedAt.Format(http.TimeFormat),
	})
}