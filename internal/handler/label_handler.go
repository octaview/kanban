package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kanban/internal/middleware"
	"kanban/internal/model"
	"kanban/internal/repository"
)

// CreateLabelRequest defines the expected request body for creating a label
type CreateLabelRequest struct {
	BoardID string `json:"board_id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Color   string `json:"color" binding:"required"`
}

// UpdateLabelRequest defines the expected request body for updating a label
type UpdateLabelRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color" binding:"required"`
}

// LabelHandler handles label-related HTTP requests
type LabelHandler struct {
	labelRepo      *repository.LabelRepository
	boardRepo      *repository.BoardRepository
	boardShareRepo *repository.BoardShareRepository
}

// NewLabelHandler creates a new LabelHandler instance
func NewLabelHandler(
	labelRepo *repository.LabelRepository,
	boardRepo *repository.BoardRepository,
	boardShareRepo *repository.BoardShareRepository,
) *LabelHandler {
	return &LabelHandler{
		labelRepo:      labelRepo,
		boardRepo:      boardRepo,
		boardShareRepo: boardShareRepo,
	}
}

// Create creates a new label
func (h *LabelHandler) Create(c *gin.Context) {
	// Get user ID from context
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

	// Parse request body
	var req CreateLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Parse board ID
	boardID, err := uuid.Parse(req.BoardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Check if board exists
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		if err == repository.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		}
		return
	}

	// Check access rights
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), boardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to create labels for this board"})
		return
	}

	// Create new label
	label := &model.Label{
		BoardID: boardID,
		Name:    req.Name,
		Color:   req.Color,
	}

	if err := h.labelRepo.Create(c.Request.Context(), label); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create label"})
		return
	}

	c.JSON(http.StatusCreated, LabelResponse{
		ID:    label.ID.String(),
		Name:  label.Name,
		Color: label.Color,
	})
}

// GetByID retrieves a label by its ID
func (h *LabelHandler) GetByID(c *gin.Context) {
	// Get user ID from context
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

	// Parse label ID
	labelIDStr := c.Param("id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	// Get label
	label, err := h.labelRepo.GetByID(c.Request.Context(), labelID)
	if err != nil {
		if err == repository.ErrLabelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve label"})
		}
		return
	}

	// Check access rights
	board, err := h.boardRepo.GetByID(c.Request.Context(), label.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), label.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this label"})
		return
	}

	c.JSON(http.StatusOK, LabelResponse{
		ID:    label.ID.String(),
		Name:  label.Name,
		Color: label.Color,
	})
}

// GetByBoardID retrieves all labels for a specific board
func (h *LabelHandler) GetByBoardID(c *gin.Context) {
	// Get user ID from context
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

	// Parse board ID
	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Check if board exists
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		if err == repository.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		}
		return
	}

	// Check access rights
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), boardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view labels for this board"})
		return
	}

	// Get labels
	labels, err := h.labelRepo.GetByBoardID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve labels"})
		return
	}

	// Prepare response
	response := make([]LabelResponse, len(labels))
	for i, label := range labels {
		response[i] = LabelResponse{
			ID:    label.ID.String(),
			Name:  label.Name,
			Color: label.Color,
		}
	}

	c.JSON(http.StatusOK, response)
}

// Update updates an existing label
func (h *LabelHandler) Update(c *gin.Context) {
	// Get user ID from context
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

	// Parse label ID
	labelIDStr := c.Param("id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	// Get existing label
	label, err := h.labelRepo.GetByID(c.Request.Context(), labelID)
	if err != nil {
		if err == repository.ErrLabelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve label"})
		}
		return
	}

	// Check access rights
	board, err := h.boardRepo.GetByID(c.Request.Context(), label.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), label.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this label"})
		return
	}

	// Parse request body
	var req UpdateLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Update label
	label.Name = req.Name
	label.Color = req.Color

	if err := h.labelRepo.Update(c.Request.Context(), label); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update label"})
		return
	}

	c.JSON(http.StatusOK, LabelResponse{
		ID:    label.ID.String(),
		Name:  label.Name,
		Color: label.Color,
	})
}

// Delete removes a label
func (h *LabelHandler) Delete(c *gin.Context) {
	// Get user ID from context
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

	// Parse label ID
	labelIDStr := c.Param("id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	// Get label
	label, err := h.labelRepo.GetByID(c.Request.Context(), labelID)
	if err != nil {
		if err == repository.ErrLabelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve label"})
		}
		return
	}

	// Check access rights
	board, err := h.boardRepo.GetByID(c.Request.Context(), label.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), label.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this label"})
		return
	}

	// Delete label
	if err := h.labelRepo.Delete(c.Request.Context(), labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label deleted successfully"})
}

// GetTasksWithLabel retrieves all tasks that have a specific label
func (h *LabelHandler) GetTasksWithLabel(c *gin.Context) {
	// Get user ID from context
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

	// Parse label ID
	labelIDStr := c.Param("id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	// Get label
	label, err := h.labelRepo.GetByID(c.Request.Context(), labelID)
	if err != nil {
		if err == repository.ErrLabelNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve label"})
		}
		return
	}

	// Check access rights
	board, err := h.boardRepo.GetByID(c.Request.Context(), label.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), label.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view tasks for this label"})
		return
	}

	// Get tasks with label
	tasks, err := h.labelRepo.GetTasksWithLabel(c.Request.Context(), labelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}

	// Prepare response (using a simplified task response for brevity)
	type TaskResponse struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ColumnID    string `json:"column_id"`
	}

	response := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		response[i] = TaskResponse{
			ID:          task.ID.String(),
			Title:       task.Title,
			Description: task.Description,
			ColumnID:    task.ColumnID.String(),
		}
	}

	c.JSON(http.StatusOK, response)
}
