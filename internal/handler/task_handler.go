package handler

import (
	"net/http"
	"time"

	"kanban/internal/middleware"
	"kanban/internal/model"
	"kanban/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TaskHandler struct {
	taskRepo       *repository.TaskRepository
	columnRepo     *repository.ColumnRepository
	boardRepo      *repository.BoardRepository
	boardShareRepo *repository.BoardShareRepository
	userRepo       *repository.UserRepository
}

func NewTaskHandler(
	taskRepo *repository.TaskRepository,
	columnRepo *repository.ColumnRepository,
	boardRepo *repository.BoardRepository,
	boardShareRepo *repository.BoardShareRepository,
	userRepo *repository.UserRepository,
) *TaskHandler {
	return &TaskHandler{
		taskRepo:       taskRepo,
		columnRepo:     columnRepo,
		boardRepo:      boardRepo,
		boardShareRepo: boardShareRepo,
		userRepo:       userRepo,
	}
}

// TaskRequest представляет запрос на создание или обновление задачи
type TaskRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	ColumnID    string     `json:"column_id" binding:"required,uuid"`
	DueDate     *time.Time `json:"due_date"`
	Position    *int       `json:"position"`
}

// TaskMoveRequest представляет запрос на перемещение задачи
type TaskMoveRequest struct {
	ColumnID string `json:"column_id" binding:"required,uuid"`
	Position int    `json:"position" binding:"required,min=0"`
}

// TaskAssignRequest представляет запрос на назначение пользователя на задачу
type TaskAssignRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

// TaskResponse представляет ответ с данными задачи
type TaskResponse struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	ColumnID     string          `json:"column_id"`
	AssignedTo   *string         `json:"assigned_to,omitempty"`
	AssigneeName *string         `json:"assignee_name,omitempty"`
	CreatedBy    string          `json:"created_by"`
	CreatorName  string          `json:"creator_name"`
	DueDate      *string         `json:"due_date,omitempty"`
	Position     int             `json:"position"`
	Labels       []LabelResponse `json:"labels,omitempty"`
}

// LabelResponse представляет ответ с данными метки
type LabelResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Create создает новую задачу
func (h *TaskHandler) Create(c *gin.Context) {
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

	// Парсим запрос
	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Проверяем существование колонки
	columnID, err := uuid.Parse(req.ColumnID)
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

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to create tasks on this board"})
		return
	}

	// Определяем позицию для новой задачи
	position := 0
	if req.Position != nil {
		position = *req.Position
	} else {
		// Если позиция не указана, добавляем в конец колонки
		tasks, err := h.taskRepo.GetByColumnID(c.Request.Context(), columnID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
			return
		}
		position = len(tasks)
	}

	// Создаем новую задачу
	task := &model.Task{
		ColumnID:    columnID,
		Title:       req.Title,
		Description: req.Description,
		CreatedBy:   authenticatedUserID,
		DueDate:     req.DueDate,
		Position:    position,
	}

	// Сохраняем задачу в БД
	if err := h.taskRepo.Create(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Находим информацию о создателе
	creator, err := h.userRepo.GetByID(c.Request.Context(), authenticatedUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user information"})
		return
	}

	// Формируем ответ
	response := TaskResponse{
		ID:          task.ID.String(),
		Title:       task.Title,
		Description: task.Description,
		ColumnID:    task.ColumnID.String(),
		CreatedBy:   task.CreatedBy.String(),
		CreatorName: creator.Name,
		Position:    task.Position,
	}

	if task.DueDate != nil {
		dueDate := task.DueDate.Format(time.RFC3339)
		response.DueDate = &dueDate
	}

	c.JSON(http.StatusCreated, response)
}

// GetByID получает задачу по ID
func (h *TaskHandler) GetByID(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this task"})
		return
	}

	// Получаем информацию о создателе
	creator, err := h.userRepo.GetByID(c.Request.Context(), task.CreatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve creator information"})
		return
	}

	// Формируем ответ
	response := TaskResponse{
		ID:          task.ID.String(),
		Title:       task.Title,
		Description: task.Description,
		ColumnID:    task.ColumnID.String(),
		CreatedBy:   task.CreatedBy.String(),
		CreatorName: creator.Name,
		Position:    task.Position,
	}

	if task.DueDate != nil {
		dueDate := task.DueDate.Format(time.RFC3339)
		response.DueDate = &dueDate
	}

	// Добавляем информацию о назначенном пользователе, если есть
	if task.AssignedTo != nil {
		assignee, err := h.userRepo.GetByID(c.Request.Context(), *task.AssignedTo)
		if err == nil {
			assignedToStr := task.AssignedTo.String()
			response.AssignedTo = &assignedToStr
			response.AssigneeName = &assignee.Name
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetByColumnID получает все задачи в колонке
func (h *TaskHandler) GetByColumnID(c *gin.Context) {
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

	// Парсим ID колонки из URL
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

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view tasks on this board"})
		return
	}

	// Получаем задачи с метками
	tasks, err := h.taskRepo.GetTasksWithLabels(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}

	// Создаем кэш пользователей для оптимизации
	userCache := make(map[uuid.UUID]*model.User)

	// Формируем ответ
	response := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		// Получаем информацию о создателе
		var creator *model.User
		var ok bool
		if creator, ok = userCache[task.CreatedBy]; !ok {
			creator, err = h.userRepo.GetByID(c.Request.Context(), task.CreatedBy)
			if err == nil {
				userCache[task.CreatedBy] = creator
			}
		}

		// Добавляем задачу в ответ
		response[i] = TaskResponse{
			ID:          task.ID.String(),
			Title:       task.Title,
			Description: task.Description,
			ColumnID:    task.ColumnID.String(),
			CreatedBy:   task.CreatedBy.String(),
			CreatorName: creator.Name,
			Position:    task.Position,
		}

		if task.DueDate != nil {
			dueDate := task.DueDate.Format(time.RFC3339)
			response[i].DueDate = &dueDate
		}

		// Добавляем информацию о назначенном пользователе, если есть
		if task.AssignedTo != nil {
			var assignee *model.User
			if assignee, ok = userCache[*task.AssignedTo]; !ok {
				assignee, err = h.userRepo.GetByID(c.Request.Context(), *task.AssignedTo)
				if err == nil {
					userCache[*task.AssignedTo] = assignee
				}
			}

			if assignee != nil {
				assignedToStr := task.AssignedTo.String()
				response[i].AssignedTo = &assignedToStr
				response[i].AssigneeName = &assignee.Name
			}
		}

		// Добавляем метки задачи
		if len(task.Labels) > 0 {
			labels := make([]LabelResponse, len(task.Labels))
			for j, label := range task.Labels {
				labels[j] = LabelResponse{
					ID:    label.ID.String(),
					Name:  label.Name,
					Color: label.Color,
				}
			}
			response[i].Labels = labels
		}
	}

	c.JSON(http.StatusOK, response)
}

// Update обновляет задачу
func (h *TaskHandler) Update(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this task"})
		return
	}

	// Парсим запрос
	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Проверяем, меняется ли колонка
	var newColumnID uuid.UUID
	var columnChanged bool
	if req.ColumnID != task.ColumnID.String() {
		columnChanged = true
		newColumnID, err = uuid.Parse(req.ColumnID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
			return
		}

		// Проверяем существование новой колонки
		newColumn, err := h.columnRepo.GetByID(c.Request.Context(), newColumnID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
			return
		}

		if newColumn == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
			return
		}

		// Проверяем, что новая колонка принадлежит той же доске
		if newColumn.BoardID != column.BoardID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot move task to a column from another board"})
			return
		}
	} else {
		newColumnID = task.ColumnID
	}

	// Обновляем задачу
	task.Title = req.Title
	task.Description = req.Description
	task.DueDate = req.DueDate

	// Если колонка изменилась или изменилась позиция
	if columnChanged || (req.Position != nil && *req.Position != task.Position) {
		position := task.Position
		if req.Position != nil {
			position = *req.Position
		}

		// Используем MoveTask для правильного обновления позиций
		if err := h.taskRepo.MoveTask(c.Request.Context(), taskID, newColumnID, position); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move task"})
			return
		}
	} else {
		// Обычное обновление без изменения позиции
		if err := h.taskRepo.Update(c.Request.Context(), task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
			return
		}
	}

	// Формируем ответ
	response := TaskResponse{
		ID:          task.ID.String(),
		Title:       task.Title,
		Description: task.Description,
		ColumnID:    newColumnID.String(),
		CreatedBy:   task.CreatedBy.String(),
		Position:    task.Position,
	}

	if task.DueDate != nil {
		dueDate := task.DueDate.Format(time.RFC3339)
		response.DueDate = &dueDate
	}

	c.JSON(http.StatusOK, response)
}

// Delete удаляет задачу
func (h *TaskHandler) Delete(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	// Разрешаем удаление только владельцу доски, создателю задачи или редакторам
	if !hasAccess && board.OwnerID != authenticatedUserID && task.CreatedBy != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this task"})
		return
	}

	// Удаляем задачу
	if err := h.taskRepo.Delete(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

// MoveTask перемещает задачу между колонками или изменяет её позицию
func (h *TaskHandler) MoveTask(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to move this task"})
		return
	}

	// Парсим запрос
	var req TaskMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Парсим ID целевой колонки
	targetColumnID, err := uuid.Parse(req.ColumnID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
		return
	}

	// Если целевая колонка отличается от текущей, проверяем её существование
	if targetColumnID != task.ColumnID {
		targetColumn, err := h.columnRepo.GetByID(c.Request.Context(), targetColumnID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve target column"})
			return
		}

		if targetColumn == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target column not found"})
			return
		}

		// Проверяем, что целевая колонка принадлежит той же доске
		if targetColumn.BoardID != column.BoardID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot move task to a column from another board"})
			return
		}
	}

	// Перемещаем задачу
	if err := h.taskRepo.MoveTask(c.Request.Context(), taskID, targetColumnID, req.Position); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task moved successfully"})
}

// AssignUser назначает пользователя на задачу
func (h *TaskHandler) AssignUser(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to assign users to this task"})
		return
	}

	// Парсим запрос
	var req TaskAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Парсим ID пользователя
	assigneeID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Проверяем существование пользователя
	assignee, err := h.userRepo.GetByID(c.Request.Context(), assigneeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	if assignee == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Назначаем пользователя на задачу
	if err := h.taskRepo.AssignUser(c.Request.Context(), taskID, assigneeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign user to task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User assigned to task successfully"})
}

// UnassignUser удаляет назначение пользователя с задачи
func (h *TaskHandler) UnassignUser(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Получаем доску, чтобы проверить права доступа
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to modify this task"})
		return
	}

	// Удаляем назначение пользователя
	if err := h.taskRepo.UnassignUser(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unassign user from task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unassigned from task successfully"})
}

// AddLabel добавляет метку к задаче
func (h *TaskHandler) AddLabel(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Парсим ID метки из URL
	labelIDStr := c.Param("label_id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	// Получаем доску, чтобы проверить права владельца
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to add labels to this task"})
		return
	}

	// Добавляем метку к задаче
	if err := h.taskRepo.AddLabel(c.Request.Context(), taskID, labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add label to task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label added to task successfully"})
}

// RemoveLabel удаляет метку с задачи
func (h *TaskHandler) RemoveLabel(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Парсим ID метки из URL
	labelIDStr := c.Param("label_id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	// Получаем доску, чтобы проверить права владельца
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to remove labels from this task"})
		return
	}

	// Удаляем метку с задачи
	if err := h.taskRepo.RemoveLabel(c.Request.Context(), taskID, labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove label from task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label removed from task successfully"})
}

// GetTaskLabels получает все метки задачи
func (h *TaskHandler) GetTaskLabels(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу с метками
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	// Получаем доску, чтобы проверить права владельца
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this task's labels"})
		return
	}

	// Получаем метки задачи
	taskWithLabels, err := h.taskRepo.GetTasksWithLabels(c.Request.Context(), column.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task labels"})
		return
	}

	// Ищем нужную задачу в списке
	var labels []LabelResponse
	for _, t := range taskWithLabels {
		if t.ID == taskID {
			for _, label := range t.Labels {
				labels = append(labels, LabelResponse{
					ID:    label.ID.String(),
					Name:  label.Name,
					Color: label.Color,
				})
			}
			break
		}
	}

	c.JSON(http.StatusOK, labels)
}

// SetDueDate устанавливает срок выполнения задачи
func (h *TaskHandler) SetDueDate(c *gin.Context) {
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

	// Парсим ID задачи из URL
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	// Получаем задачу
	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	// Получаем колонку, чтобы найти ID доски
	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	// Проверяем права доступа
	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	// Получаем доску, чтобы проверить права владельца
	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to modify this task"})
		return
	}

	// Парсим запрос
	var req struct {
		DueDate *time.Time `json:"due_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Обновляем срок выполнения задачи
	task.DueDate = req.DueDate
	if err := h.taskRepo.Update(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task due date"})
		return
	}

	// Формируем ответ
	response := TaskResponse{
		ID:          task.ID.String(),
		Title:       task.Title,
		Description: task.Description,
		ColumnID:    task.ColumnID.String(),
		CreatedBy:   task.CreatedBy.String(),
		Position:    task.Position,
	}

	if task.DueDate != nil {
		dueDate := task.DueDate.Format(time.RFC3339)
		response.DueDate = &dueDate
	}

	c.JSON(http.StatusOK, response)
}
