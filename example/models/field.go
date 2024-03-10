package models

import "time"

type Field struct {
	ID         uint64 `gorm:"primaryKey"`
	DocumentID uint64
	Key        string
	Value      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (*Field) TableName() string {
	return "docs.documents_fields"
}
