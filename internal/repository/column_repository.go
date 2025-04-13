package repository

import (
	"context"
	"errors"
	"kanban/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ColumnRepository struct {
	db *gorm.DB
}

func NewColumnRepository(db *gorm.DB) *ColumnRepository {
	return &ColumnRepository{db: db}
}

func (r *ColumnRepository) Create(ctx context.Context, column *model.Column) error {
	return r.db.WithContext(ctx).Create(column).Error
}

func (r *ColumnRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Column, error) {
	var column model.Column
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&column).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &column, nil
}

func (r *ColumnRepository) GetByBoardID(ctx context.Context, boardID uuid.UUID) ([]model.Column, error) {
	var columns []model.Column
	err := r.db.WithContext(ctx).Where("board_id = ?", boardID).Order("position").Find(&columns).Error
	return columns, err
}

func (r *ColumnRepository) Update(ctx context.Context, column *model.Column) error {
	return r.db.WithContext(ctx).Save(column).Error
}

func (r *ColumnRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Column{}, id).Error
}

func (r *ColumnRepository) GetMaxPosition(ctx context.Context, boardID uuid.UUID) (int, error) {
	var maxPosition struct {
		Max int
	}
	err := r.db.WithContext(ctx).Model(&model.Column{}).
		Select("COALESCE(MAX(position), 0) as max").
		Where("board_id = ?", boardID).
		Scan(&maxPosition).Error

	return maxPosition.Max, err
}

func (r *ColumnRepository) ReorderColumns(ctx context.Context, columns []model.Column) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, column := range columns {
			if err := tx.Model(&model.Column{}).Where("id = ?", column.ID).
				Update("position", column.Position).Error; err != nil {
				return err
			}
		}
		return nil
	})
}