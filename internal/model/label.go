package model

import (
	"github.com/google/uuid"
)

type Label struct {
	ID      uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	BoardID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name    string    `gorm:"not null"`
	Color   string    `gorm:"not null"`

	Board Board `gorm:"foreignKey:BoardID"`
	Tasks []Task `gorm:"many2many:task_labels"`
}