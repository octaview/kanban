package model

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	ColumnID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Title       string     `gorm:"not null"`
	Description string
	AssignedTo  *uuid.UUID `gorm:"type:uuid"`
	CreatedBy   uuid.UUID  `gorm:"type:uuid;not null"`
	DueDate     *time.Time
	Position    int        `gorm:"not null"`

	Column     Column `gorm:"foreignKey:ColumnID"`
	Assignee   User   `gorm:"foreignKey:AssignedTo"`
	Creator    User   `gorm:"foreignKey:CreatedBy"`
	Labels     []Label `gorm:"many2many:task_labels"`
}