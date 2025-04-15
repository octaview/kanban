package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kanban/internal/model"
)

var (
	ErrLabelNotFound = errors.New("label not found")
)

type LabelRepository struct {
	db *gorm.DB
}

func NewLabelRepository(db *gorm.DB) *LabelRepository {
	return &LabelRepository{db: db}
}

// Create adds a new label to the database
func (r *LabelRepository) Create(ctx context.Context, label *model.Label) error {
	return r.db.WithContext(ctx).Create(label).Error
}

// GetByID retrieves a label by its ID
func (r *LabelRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Label, error) {
	var label model.Label
	result := r.db.WithContext(ctx).First(&label, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrLabelNotFound
		}
		return nil, result.Error
	}
	return &label, nil
}

// GetByBoardID retrieves all labels for a specific board
func (r *LabelRepository) GetByBoardID(ctx context.Context, boardID uuid.UUID) ([]model.Label, error) {
	var labels []model.Label
	result := r.db.WithContext(ctx).Where("board_id = ?", boardID).Find(&labels)
	if result.Error != nil {
		return nil, result.Error
	}
	return labels, nil
}

// GetByTaskID retrieves all labels associated with a specific task
func (r *LabelRepository) GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]model.Label, error) {
	var labels []model.Label
	result := r.db.WithContext(ctx).
		Joins("JOIN task_labels ON task_labels.label_id = labels.id").
		Where("task_labels.task_id = ?", taskID).
		Find(&labels)
	
	if result.Error != nil {
		return nil, result.Error
	}
	return labels, nil
}

// Update updates an existing label
func (r *LabelRepository) Update(ctx context.Context, label *model.Label) error {
	result := r.db.WithContext(ctx).Save(label)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLabelNotFound
	}
	return nil
}

// Delete removes a label by its ID
func (r *LabelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Label{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLabelNotFound
	}
	return nil
}

// AttachToTask adds a label to a specific task
func (r *LabelRepository) AttachToTask(ctx context.Context, labelID, taskID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO task_labels (label_id, task_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		labelID, taskID,
	).Error
}

// DetachFromTask removes a label from a specific task
func (r *LabelRepository) DetachFromTask(ctx context.Context, labelID, taskID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM task_labels WHERE label_id = ? AND task_id = ?",
		labelID, taskID,
	).Error
}

// GetTasksWithLabel retrieves all tasks that have a specific label
func (r *LabelRepository) GetTasksWithLabel(ctx context.Context, labelID uuid.UUID) ([]model.Task, error) {
	var tasks []model.Task
	result := r.db.WithContext(ctx).
		Joins("JOIN task_labels ON task_labels.task_id = tasks.id").
		Where("task_labels.label_id = ?", labelID).
		Find(&tasks)
	
	if result.Error != nil {
		return nil, result.Error
	}
	return tasks, nil
}