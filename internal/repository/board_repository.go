package repository

import (
	"context"
	"errors"
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

func (r *BoardRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Board, error) {
	var board model.Board
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&board).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil to indicate that the board was not found
		}
		return nil, err
	}
	return &board, nil
}

func (r *BoardRepository) Update(ctx context.Context, board *model.Board) error {
	return r.db.WithContext(ctx).Save(board).Error
}