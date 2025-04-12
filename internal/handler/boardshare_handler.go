package handler

import (
	"net/http"

	"kanban/internal/middleware"
	"kanban/internal/model"
	"kanban/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BoardShareHandler struct {
	boardRepo      *repository.BoardRepository
	userRepo       *repository.UserRepository
	boardShareRepo *repository.BoardShareRepository
}

func NewBoardShareHandler(
	boardRepo *repository.BoardRepository,
	userRepo *repository.UserRepository,
	boardShareRepo *repository.BoardShareRepository,
) *BoardShareHandler {
	return &BoardShareHandler{
		boardRepo:      boardRepo,
		userRepo:       userRepo,
		boardShareRepo: boardShareRepo,
	}
}

// ShareBoardRequest представляет запрос на предоставление доступа к доске
type ShareBoardRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=viewer editor"`
}

// BoardShareResponse представляет информацию о пользователе с доступом к доске
type BoardShareResponse struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	IsOwner   bool   `json:"is_owner"`
}

// ShareBoard предоставляет доступ к доске по email пользователя
func (h *BoardShareHandler) ShareBoard(c *gin.Context) {
	// Получаем ID текущего пользователя из контекста
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

	// Парсим ID доски из URL
	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Получаем доску
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		return
	}

	// Проверяем, является ли пользователь владельцем доски
	if board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the board owner can share the board"})
		return
	}

	// Парсим запрос
	var req ShareBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Находим пользователя по email
	targetUser, err := h.userRepo.FindByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		return
	}

	if targetUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Нельзя поделиться с самим собой
	if targetUser.ID == authenticatedUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot share board with yourself"})
		return
	}

	// Предоставляем доступ
	if err := h.boardShareRepo.ShareBoard(c.Request.Context(), boardID, targetUser.ID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share board"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Board shared successfully",
		"share": BoardShareResponse{
			UserID:  targetUser.ID.String(),
			Email:   targetUser.Email,
			Name:    targetUser.Name,
			Role:    req.Role,
			IsOwner: false,
		},
	})
}

// RemoveShare удаляет доступ пользователя к доске
func (h *BoardShareHandler) RemoveShare(c *gin.Context) {
	// Получаем ID текущего пользователя из контекста
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

	// Парсим ID доски из URL
	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Парсим ID пользователя для удаления из URL
	targetUserIDStr := c.Param("user_id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Получаем доску
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		return
	}

	// Проверяем, является ли пользователь владельцем доски
	if board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the board owner can remove access"})
		return
	}

	// Удаляем доступ
	if err := h.boardShareRepo.RemoveShare(c.Request.Context(), boardID, targetUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove share"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Board access removed successfully"})
}

// GetBoardShares возвращает список пользователей с доступом к доске
func (h *BoardShareHandler) GetBoardShares(c *gin.Context) {
	// Получаем ID текущего пользователя из контекста
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

	// Парсим ID доски из URL
	boardIDStr := c.Param("id")
	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID format"})
		return
	}

	// Получаем доску
	board, err := h.boardRepo.GetByID(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Board not found"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), boardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this board"})
		return
	}

	// Получаем список пользователей с доступом
	shares, err := h.boardShareRepo.GetBoardShares(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board shares"})
		return
	}

	// Формируем ответ
	response := make([]BoardShareResponse, 0, len(shares)+1)

	// Добавляем владельца
	if board.OwnerID == authenticatedUserID {
		response = append(response, BoardShareResponse{
			UserID:  authenticatedUserID.String(),
			Email:   c.GetString("user_email"),
			Name:    c.GetString("user_name"),
			Role:    "owner",
			IsOwner: true,
		})
	}

	// Добавляем остальных пользователей
	for _, share := range shares {
		response = append(response, BoardShareResponse{
			UserID:  share.UserID.String(),
			Email:   share.User.Email,
			Name:    share.User.Name,
			Role:    share.Role,
			IsOwner: false,
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetSharedBoards возвращает список досок, к которым у пользователя есть доступ
func (h *BoardShareHandler) GetSharedBoards(c *gin.Context) {
	// Получаем ID текущего пользователя из контекста
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

	// Получаем доски с доступом
	boards, err := h.boardShareRepo.GetSharedBoards(c.Request.Context(), authenticatedUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve shared boards"})
		return
	}

	// Формируем ответ
	response := make([]BoardResponse, len(boards))
	for i, board := range boards {
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