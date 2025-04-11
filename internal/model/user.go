package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Email          string    `gorm:"uniqueIndex;not null"`
	HashedPassword string    `gorm:"not null"`
	Name           string    `gorm:"not null"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}
