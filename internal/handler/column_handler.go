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

func (h *ColumnHandler) checkBoardAccess(c *gin.Context, boardID uuid.UUID, userID uuid.UUID, requiredRole string) (bool, error) {
    board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
    if err != nil {
        return false, err
    }

    if board == nil {
        return false, nil
    }

    // Делегируем проверку прав доступа репозиторию
    return h.boardShareRepo.CheckAccess(c.Request.Context(), boardID, userID, requiredRole)
}

// Create создает новую колонку на доске
func (h *ColumnHandler) Create(c *gin.Context) {
	// Получаем ID пользователя из контекста
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

	// Парсим запрос
	var req CreateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Парсим ID доски
	boardID, err := uuid.Parse(req.BoardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Проверяем права доступа к доске (требуется роль редактора)
	hasAccess, err := h.checkBoardAccess(c, boardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to add columns to this board"})
		return
	}

	// Определяем позицию для новой колонки
	position := req.Position
	if position == 0 {
		// Если позиция не указана, добавляем в конец
		maxPosition, err := h.columnRepo.GetMaxPosition(c.Request.Context(), boardID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to determine column position"})
			return
		}
		position = maxPosition + 1
	}

	// Создаем новую колонку
	column := &model.Column{
		BoardID:  boardID,
		Title:    req.Title,
		Position: position,
	}

	if err := h.columnRepo.Create(c.Request.Context(), column); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create column"})
		return
	}

	// Возвращаем созданную колонку
	c.JSON(http.StatusCreated, ColumnResponse{
		ID:       column.ID.String(),
		BoardID:  column.BoardID.String(),
		Title:    column.Title,
		Position: column.Position,
	})
}

// GetAll возвращает все колонки указанной доски
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

	// Проверяем права на просмотр доски
	hasAccess, err := h.checkBoardAccess(c, boardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this board"})
		return
	}

	// Получаем все колонки доски, отсортированные по позиции
	columns, err := h.columnRepo.GetByBoardID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve columns"})
		return
	}

	// Формируем ответ
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

// GetByID возвращает колонку по ID
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

	// Получаем колонку
	column, err := h.columnRepo.GetByID(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	if column == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
		return
	}

	// Проверяем права доступа к доске, которой принадлежит колонка
	hasAccess, err := h.checkBoardAccess(c, column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this column"})
		return
	}

	// Возвращаем информацию о колонке
	c.JSON(http.StatusOK, ColumnResponse{
		ID:       column.ID.String(),
		BoardID:  column.BoardID.String(),
		Title:    column.Title,
		Position: column.Position,
	})
}

// Update обновляет данные колонки
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

	// Получаем колонку
	column, err := h.columnRepo.GetByID(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	if column == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
		return
	}

	// Проверяем права на редактирование доски
	hasAccess, err := h.checkBoardAccess(c, column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this column"})
		return
	}

	// Парсим запрос
	var req UpdateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Обновляем только предоставленные поля
	if req.Title != "" {
		column.Title = req.Title
	}
	if req.Position != 0 {
		column.Position = req.Position
	}

	// Сохраняем изменения
	if err := h.columnRepo.Update(c.Request.Context(), column); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update column"})
		return
	}

	// Возвращаем обновленную колонку
	c.JSON(http.StatusOK, ColumnResponse{
		ID:       column.ID.String(),
		BoardID:  column.BoardID.String(),
		Title:    column.Title,
		Position: column.Position,
	})
}

// Delete удаляет колонку
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

	// Получаем колонку
	column, err := h.columnRepo.GetByID(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	if column == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
		return
	}

	// Проверяем права на редактирование доски
	hasAccess, err := h.checkBoardAccess(c, column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this column"})
		return
	}

	// Удаляем колонку
	if err := h.columnRepo.Delete(c.Request.Context(), columnID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete column"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Column deleted successfully"})
}

// ReorderColumns изменяет порядок колонок на доске
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

	// Проверяем права на редактирование доски
	hasAccess, err := h.checkBoardAccess(c, boardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to reorder columns on this board"})
		return
	}

	// Парсим запрос
	var req ReorderColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Подготавливаем массив колонок для обновления
	columns := make([]model.Column, len(req.Columns))
	columnIDs := make([]uuid.UUID, len(req.Columns))

	// Сначала собираем все ID колонок для проверки
	for i, col := range req.Columns {
		columnID, err := uuid.Parse(col.ID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
			return
		}
		columnIDs[i] = columnID
	}

	// Проверяем, что все колонки существуют и принадлежат указанной доске
	existingColumns, err := h.columnRepo.GetByIDs(c.Request.Context(), columnIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve columns"})
		return
	}

	if len(existingColumns) != len(columnIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some columns not found"})
		return
	}

	// Проверяем, что все колонки принадлежат указанной доске
	for _, column := range existingColumns {
		if column.BoardID != boardID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All columns must belong to the specified board"})
			return
		}
	}

	// Формируем массив колонок с новыми позициями
	for i, col := range req.Columns {
		columnID, _ := uuid.Parse(col.ID) // Парсинг уже проверен выше
		columns[i] = model.Column{
			ID:       columnID,
			Position: col.Position,
			BoardID:  boardID,
		}
	}

	// Обновляем позиции колонок
	if err := h.columnRepo.ReorderColumns(c.Request.Context(), columns); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder columns"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Columns reordered successfully"})
}