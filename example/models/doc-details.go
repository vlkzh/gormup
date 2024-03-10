package models

import (
	"time"

	"gorm.io/gorm"
)

type DocDetails struct {
	ID uint64 `gorm:"primaryKey"`

	Printable bool
	Published bool
	Comment   string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (*DocDetails) TableName() string {
	return "docs.documents_details"
}
