package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kanban/internal/model"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create adds a new task to the database
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID retrieves a task by its ID
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	var task model.Task
	result := r.db.WithContext(ctx).First(&task, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, result.Error
	}
	return &task, nil
}

// GetByColumnID retrieves all tasks in a specific column
func (r *TaskRepository) GetByColumnID(ctx context.Context, columnID uuid.UUID) ([]model.Task, error) {
	var tasks []model.Task
	result := r.db.WithContext(ctx).Where("column_id = ?", columnID).Order("position").Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}
	return tasks, nil
}

// GetTasksWithLabels retrieves tasks with their associated labels
func (r *TaskRepository) GetTasksWithLabels(ctx context.Context, columnID uuid.UUID) ([]model.Task, error) {
	var tasks []model.Task
	result := r.db.WithContext(ctx).
		Preload("Labels").
		Where("column_id = ?", columnID).
		Order("position").
		Find(&tasks)
	
	if result.Error != nil {
		return nil, result.Error
	}
	return tasks, nil
}

// Update updates an existing task
func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	result := r.db.WithContext(ctx).Save(task)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// Delete removes a task by its ID
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Task{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// MoveTask updates the position and/or column of a task
func (r *TaskRepository) MoveTask(ctx context.Context, taskID uuid.UUID, columnID uuid.UUID, newPosition int) error {
	// Start a transaction
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get the task
		var task model.Task
		if err := tx.First(&task, "id = ?", taskID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTaskNotFound
			}
			return err
		}

		oldColumnID := task.ColumnID
		oldPosition := task.Position

		// If moving to a different column
		if oldColumnID != columnID {
			// Adjust positions in the old column (decrement positions of tasks after this one)
			if err := tx.Model(&model.Task{}).
				Where("column_id = ? AND position > ?", oldColumnID, oldPosition).
				Update("position", gorm.Expr("position - 1")).Error; err != nil {
				return err
			}

			// Make space in the new column (increment positions of tasks at or after the target position)
			if err := tx.Model(&model.Task{}).
				Where("column_id = ? AND position >= ?", columnID, newPosition).
				Update("position", gorm.Expr("position + 1")).Error; err != nil {
				return err
			}

			// Update the task's column and position
			task.ColumnID = columnID
			task.Position = newPosition
		} else if oldPosition != newPosition {
			// Moving within the same column
			if oldPosition < newPosition {
				// Moving down: decrement positions of tasks between old and new
				if err := tx.Model(&model.Task{}).
					Where("column_id = ? AND position > ? AND position <= ?", columnID, oldPosition, newPosition).
					Update("position", gorm.Expr("position - 1")).Error; err != nil {
					return err
				}
			} else {
				// Moving up: increment positions of tasks between new and old
				if err := tx.Model(&model.Task{}).
					Where("column_id = ? AND position >= ? AND position < ?", columnID, newPosition, oldPosition).
					Update("position", gorm.Expr("position + 1")).Error; err != nil {
					return err
				}
			}

			// Update the task's position
			task.Position = newPosition
		}

		// Save the updated task
		return tx.Save(&task).Error
	})
}

// AddLabel adds a label to a task
func (r *TaskRepository) AddLabel(ctx context.Context, taskID, labelID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO task_labels (task_id, label_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		taskID, labelID,
	).Error
}

// RemoveLabel removes a label from a task
func (r *TaskRepository) RemoveLabel(ctx context.Context, taskID, labelID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM task_labels WHERE task_id = ? AND label_id = ?",
		taskID, labelID,
	).Error
}

// AssignUser assigns a user to a task
func (r *TaskRepository) AssignUser(ctx context.Context, taskID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("id = ?", taskID).
		Update("assigned_to", userID)
	
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

// UnassignUser removes user assignment from a task
func (r *TaskRepository) UnassignUser(ctx context.Context, taskID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("id = ?", taskID).
		Update("assigned_to", nil)
	
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}