package model

import (
	"time"

	"github.com/google/uuid"
)

type Board struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Title       string    `gorm:"not null"`
	Description string
	OwnerID     uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Owner User `gorm:"foreignKey:OwnerID"`
}
