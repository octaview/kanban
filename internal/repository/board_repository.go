package repository

import (
	"context"
	"kanban/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BoardRepository struct {
	db *gorm.DB
}

func NewBoardRepository(db *gorm.DB) *BoardRepository {
	return &BoardRepository{db: db}
}

func (r *BoardRepository) Create(ctx context.Context, board *model.Board) error {
	return r.db.WithContext(ctx).Create(board).Error
}

func (r *BoardRepository) GetOwned(ctx context.Context, ownerID uuid.UUID) ([]model.Board, error) {
	var boards []model.Board
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&boards).Error
	return boards, err
}

func (r *BoardRepository) CountOwned(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Board{}).Where("owner_id = ?", ownerID).Count(&count).Error
	return count, err
}
