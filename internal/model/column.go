package model

import (
	"github.com/google/uuid"
)

type Column struct {
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	BoardID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Title    string    `gorm:"not null"`
	Position int       `gorm:"not null"`

	Board Board `gorm:"foreignKey:BoardID"`
}
