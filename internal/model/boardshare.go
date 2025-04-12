package model

import (
	"time"

	"github.com/google/uuid"
)

// BoardShare представляет связь между пользователем и доской
type BoardShare struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	BoardID   uuid.UUID `gorm:"type:uuid;not null;index"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Role      string    `gorm:"not null;check:role IN ('viewer', 'editor')"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	Board Board `gorm:"foreignKey:BoardID"`
	User  User  `gorm:"foreignKey:UserID"`
}

// Роли пользователей для доски
const (
	RoleViewer = "viewer" // может только просматривать
	RoleEditor = "editor" // может редактировать
)