package handler

import (
	"net/http"

	"kanban/internal/middleware"
	"kanban/internal/model"
	"kanban/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ColumnHandler struct {
	columnRepo     *repository.ColumnRepository
	boardRepo      *repository.BoardRepository
	boardShareRepo *repository.BoardShareRepository
}

func NewColumnHandler(columnRepo *repository.ColumnRepository, boardRepo *repository.BoardRepository, boardShareRepo *repository.BoardShareRepository) *ColumnHandler {
	return &ColumnHandler{
		columnRepo:     columnRepo,
		boardRepo:      boardRepo,
		boardShareRepo: boardShareRepo,
	}
}

type CreateColumnRequest struct {
	Title    string `json:"title" binding:"required"`
	BoardID  string `json:"board_id" binding:"required"`
	Position int    `json:"position"`
}

type UpdateColumnRequest struct {
	Title    string `json:"title"`
	Position int    `json:"position"`
}

type ColumnResponse struct {
	ID       string `json:"id"`
	BoardID  string `json:"board_id"`
	Title    string `json:"title"`
	Position int    `json:"position"`
}

type ReorderColumnsRequest struct {
	Columns []struct {
		ID       string `json:"id" binding:"required"`
		Position int    `json:"position" binding:"required"`
	} `json:"columns" binding:"required"`
}

func (h *ColumnHandler) checkBoardAccess(c *gin.Context, boardID uuid.UUID, userID uuid.UUID, requiredRole model.Role) (bool, error) {
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		return false, err
	}

	if board == nil {
		return false, nil
	}

	if board.OwnerID == userID {
		return true, nil
	}

	return h.boardShareRepo.CheckAccess(c.Request.Context(), boardID, userID, requiredRole)
}

func (h *ColumnHandler) Create(c *gin.Context) {
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

	var req CreateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	boardID, err := uuid.Parse(req.BoardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	hasAccess, err := h.checkBoardAccess(c, boardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to add columns to this board"})
		return
	}

	position := req.Position
	if position == 0 {
		maxPosition, err := h.columnRepo.GetMaxPosition(c.Request.Context(), boardID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to determine column position"})
			return
		}
		position = maxPosition + 1
	}

	column := &model.Column{
		BoardID:  boardID,
		Title:    req.Title,
		Position: position,
	}

	if err := h.columnRepo.Create(c.Request.Context(), column); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create column"})
		return
	}

	c.JSON(http.StatusCreated, ColumnResponse{
		ID:       column.ID.String(),
		BoardID:  column.BoardID.String(),
		Title:    column.Title,
		Position: column.Position,
	})
}

func (h *ColumnHandler) GetAll(c *gin.Context) {
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

	boardIDStr := c.Param("board_id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	hasAccess, err := h.checkBoardAccess(c, boardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this board"})
		return
	}

	columns, err := h.columnRepo.GetByBoardID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve columns"})
		return
	}

	response := make([]ColumnResponse, len(columns))
	for i, column := range columns {
		response[i] = ColumnResponse{
			ID:       column.ID.String(),
			BoardID:  column.BoardID.String(),
			Title:    column.Title,
			Position: column.Position,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *ColumnHandler) GetByID(c *gin.Context) {
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

	columnIDStr := c.Param("id")
	columnID, err := uuid.Parse(columnIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	if column == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
		return
	}

	hasAccess, err := h.checkBoardAccess(c, column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this column"})
		return
	}

	c.JSON(http.StatusOK, ColumnResponse{
		ID:       column.ID.String(),
		BoardID:  column.BoardID.String(),
		Title:    column.Title,
		Position: column.Position,
	})
}

func (h *ColumnHandler) Update(c *gin.Context) {
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

	columnIDStr := c.Param("id")
	columnID, err := uuid.Parse(columnIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	if column == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
		return
	}

	hasAccess, err := h.checkBoardAccess(c, column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this column"})
		return
	}

	var req UpdateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Title != "" {
		column.Title = req.Title
	}
	if req.Position != 0 {
		column.Position = req.Position
	}

	if err := h.columnRepo.Update(c.Request.Context(), column); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update column"})
		return
	}

	c.JSON(http.StatusOK, ColumnResponse{
		ID:       column.ID.String(),
		BoardID:  column.BoardID.String(),
		Title:    column.Title,
		Position: column.Position,
	})
}

func (h *ColumnHandler) Delete(c *gin.Context) {
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

	columnIDStr := c.Param("id")
	columnID, err := uuid.Parse(columnIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	if column == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
		return
	}

	hasAccess, err := h.checkBoardAccess(c, column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this column"})
		return
	}

	if err := h.columnRepo.Delete(c.Request.Context(), columnID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete column"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Column deleted successfully"})
}

func (h *ColumnHandler) ReorderColumns(c *gin.Context) {
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

	boardIDStr := c.Param("board_id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	hasAccess, err := h.checkBoardAccess(c, boardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to reorder columns on this board"})
		return
	}

	var req ReorderColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	columns := make([]model.Column, len(req.Columns))
	for i, col := range req.Columns {
		columnID, err := uuid.Parse(col.ID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
			return
		}

		column, err := h.columnRepo.GetByID(c.Request.Context(), columnID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
			return
		}

		if column == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
			return
		}

		if column.BoardID != boardID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All columns must belong to the specified board"})
			return
		}

		columns[i] = model.Column{
			ID:       columnID,
			Position: col.Position,
			BoardID:  boardID,
		}
	}

	if err := h.columnRepo.ReorderColumns(c.Request.Context(), columns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder columns"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Columns reordered successfully"})
}