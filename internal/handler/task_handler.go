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

type TaskRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	ColumnID    string     `json:"column_id" binding:"required,uuid"`
	DueDate     *time.Time `json:"due_date"`
	Position    *int       `json:"position"`
}

type TaskMoveRequest struct {
	ColumnID string `json:"column_id" binding:"required,uuid"`
	Position int    `json:"position" binding:"required,min=0"`
}

type TaskAssignRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

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

func (h *TaskHandler) Create(c *gin.Context) {
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

	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

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

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to create tasks on this board"})
		return
	}

	position := 0
	if req.Position != nil {
		position = *req.Position
	} else {
		tasks, err := h.taskRepo.GetByColumnID(c.Request.Context(), columnID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
			return
		}
		position = len(tasks)
	}

	task := &model.Task{
		ColumnID:    columnID,
		Title:       req.Title,
		Description: req.Description,
		CreatedBy:   authenticatedUserID,
		DueDate:     req.DueDate,
		Position:    position,
	}

	if err := h.taskRepo.Create(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	creator, err := h.userRepo.GetByID(c.Request.Context(), authenticatedUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user information"})
		return
	}

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

func (h *TaskHandler) GetByID(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this task"})
		return
	}

	creator, err := h.userRepo.GetByID(c.Request.Context(), task.CreatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve creator information"})
		return
	}

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

func (h *TaskHandler) GetByColumnID(c *gin.Context) {
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

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view tasks on this board"})
		return
	}

	tasks, err := h.taskRepo.GetTasksWithLabels(c.Request.Context(), columnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}

	userCache := make(map[uuid.UUID]*model.User)

	response := make([]TaskResponse, len(tasks))
	for i, task := range tasks {
		var creator *model.User
		var ok bool
		if creator, ok = userCache[task.CreatedBy]; !ok {
			creator, err = h.userRepo.GetByID(c.Request.Context(), task.CreatedBy)
			if err == nil {
				userCache[task.CreatedBy] = creator
			}
		}

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

func (h *TaskHandler) Update(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this task"})
		return
	}

	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var newColumnID uuid.UUID
	var columnChanged bool
	if req.ColumnID != task.ColumnID.String() {
		columnChanged = true
		newColumnID, err = uuid.Parse(req.ColumnID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
			return
		}

		newColumn, err := h.columnRepo.GetByID(c.Request.Context(), newColumnID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
			return
		}

		if newColumn == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Column not found"})
			return
		}

		if newColumn.BoardID != column.BoardID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot move task to a column from another board"})
			return
		}
	} else {
		newColumnID = task.ColumnID
	}

	task.Title = req.Title
	task.Description = req.Description
	task.DueDate = req.DueDate

	if columnChanged || (req.Position != nil && *req.Position != task.Position) {
		position := task.Position
		if req.Position != nil {
			position = *req.Position
		}

		if err := h.taskRepo.MoveTask(c.Request.Context(), taskID, newColumnID, position); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move task"})
			return
		}
	} else {
		if err := h.taskRepo.Update(c.Request.Context(), task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
			return
		}
	}

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

func (h *TaskHandler) Delete(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID && task.CreatedBy != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this task"})
		return
	}

	if err := h.taskRepo.Delete(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

func (h *TaskHandler) MoveTask(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to move this task"})
		return
	}

	var req TaskMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	targetColumnID, err := uuid.Parse(req.ColumnID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID format"})
		return
	}

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

		if targetColumn.BoardID != column.BoardID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot move task to a column from another board"})
			return
		}
	}

	if err := h.taskRepo.MoveTask(c.Request.Context(), taskID, targetColumnID, req.Position); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task moved successfully"})
}

func (h *TaskHandler) AssignUser(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to assign users to this task"})
		return
	}

	var req TaskAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	assigneeID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	assignee, err := h.userRepo.GetByID(c.Request.Context(), assigneeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	if assignee == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := h.taskRepo.AssignUser(c.Request.Context(), taskID, assigneeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign user to task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User assigned to task successfully"})
}

func (h *TaskHandler) UnassignUser(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to modify this task"})
		return
	}

	if err := h.taskRepo.UnassignUser(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unassign user from task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unassigned from task successfully"})
}

func (h *TaskHandler) AddLabel(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	labelIDStr := c.Param("label_id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to add labels to this task"})
		return
	}

	if err := h.taskRepo.AddLabel(c.Request.Context(), taskID, labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add label to task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label added to task successfully"})
}

func (h *TaskHandler) RemoveLabel(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	labelIDStr := c.Param("label_id")
	labelID, err := uuid.Parse(labelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to remove labels from this task"})
		return
	}

	if err := h.taskRepo.RemoveLabel(c.Request.Context(), taskID, labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove label from task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label removed from task successfully"})
}

func (h *TaskHandler) GetTaskLabels(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleViewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this task's labels"})
		return
	}

	taskWithLabels, err := h.taskRepo.GetTasksWithLabels(c.Request.Context(), column.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task labels"})
		return
	}

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

func (h *TaskHandler) SetDueDate(c *gin.Context) {
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

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task"})
		}
		return
	}

	column, err := h.columnRepo.GetByID(c.Request.Context(), task.ColumnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve column"})
		return
	}

	hasAccess, err := h.boardShareRepo.CheckAccess(c.Request.Context(), column.BoardID, authenticatedUserID, model.RoleEditor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}

	board, err := h.boardRepo.GetByID(c.Request.Context(), column.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board"})
		return
	}

	if !hasAccess && board.OwnerID != authenticatedUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to modify this task"})
		return
	}

	var req struct {
		DueDate *time.Time `json:"due_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	task.DueDate = req.DueDate
	if err := h.taskRepo.Update(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task due date"})
		return
	}

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
