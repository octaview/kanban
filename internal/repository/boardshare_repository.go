package repository

import (
	"context"
	"errors"
	"kanban/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BoardShareRepository struct {
	db *gorm.DB
}

func NewBoardShareRepository(db *gorm.DB) *BoardShareRepository {
	return &BoardShareRepository{db: db}
}

// ShareBoard добавляет пользователя к доске с указанной ролью
func (r *BoardShareRepository) ShareBoard(ctx context.Context, boardID, userID uuid.UUID, role string) error {
	share := model.BoardShare{
		BoardID: boardID,
		UserID:  userID,
		Role:    role,
	}
	
	// Используем транзакцию для предотвращения гонок
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Проверяем, существует ли уже доступ
		var existingShare model.BoardShare
		err := tx.Where("board_id = ? AND user_id = ?", boardID, userID).First(&existingShare).Error
		
		// Если запись уже существует, обновляем роль
		if err == nil {
			existingShare.Role = role
			return tx.Save(&existingShare).Error
		}
		
		// Иначе, если ошибка не связана с отсутствием записи, возвращаем ее
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		
		// Если запись не существует, создаем новую
		return tx.Create(&share).Error
	})
}

// RemoveShare удаляет доступ пользователя к доске
func (r *BoardShareRepository) RemoveShare(ctx context.Context, boardID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("board_id = ? AND user_id = ?", boardID, userID).Delete(&model.BoardShare{}).Error
}

// GetBoardShares возвращает список пользователей с доступом к доске
func (r *BoardShareRepository) GetBoardShares(ctx context.Context, boardID uuid.UUID) ([]model.BoardShare, error) {
	var shares []model.BoardShare
	
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("board_id = ?", boardID).
		Find(&shares).Error
	
	return shares, err
}

// GetSharedBoards возвращает доски, к которым пользователь имеет доступ
func (r *BoardShareRepository) GetSharedBoards(ctx context.Context, userID uuid.UUID) ([]model.Board, error) {
	var boards []model.Board
	
	err := r.db.WithContext(ctx).
		Joins("JOIN board_shares ON board_shares.board_id = boards.id").
		Where("board_shares.user_id = ?", userID).
		Find(&boards).Error
	
	return boards, err
}

// GetUserRole возвращает роль пользователя для доски (или пустую строку, если нет доступа)
func (r *BoardShareRepository) GetUserRole(ctx context.Context, boardID, userID uuid.UUID) (string, error) {
	var share model.BoardShare
	
	err := r.db.WithContext(ctx).
		Where("board_id = ? AND user_id = ?", boardID, userID).
		First(&share).Error
	
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil // Пользователь не имеет доступа
	}
	
	if err != nil {
		return "", err
	}
	
	return share.Role, nil
}

// CheckAccess проверяет, имеет ли пользователь доступ к доске с указанной ролью или выше
func (r *BoardShareRepository) CheckAccess(ctx context.Context, boardID, userID uuid.UUID, requiredRole string) (bool, error) {
	// Проверяем, является ли пользователь владельцем
	var board model.Board
	err := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", boardID, userID).
		First(&board).Error
	
	// Владелец всегда имеет полный доступ
	if err == nil {
		return true, nil
	}
	
	// Если ошибка не связана с отсутствием записи, возвращаем ее
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	
	// Проверяем права по таблице доступа
	var share model.BoardShare
	err = r.db.WithContext(ctx).
		Where("board_id = ? AND user_id = ?", boardID, userID).
		First(&share).Error
	
	// Нет доступа
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	
	if err != nil {
		return false, err
	}
	
	// Если требуется роль "viewer", то подойдет любая роль
	if requiredRole == model.RoleViewer {
		return true, nil
	}
	
	// Если требуется роль "editor", то проверяем что у пользователя роль "editor"
	return share.Role == model.RoleEditor, nil
}