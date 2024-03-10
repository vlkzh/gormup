package models

import (
	"time"

	"gorm.io/gorm"
)

type DocType struct {
	Type string
	Code string
}

type Document struct {
	ID      uint64  `gorm:"primaryKey"`
	DocType DocType `gorm:"embedded;embeddedPrefix:doc_"`

	DetailID uint64

	Name string
	Desc string

	Number int64
	Price  float64

	Transacted   bool
	TransactedAt *time.Time

	ContactInfo ContactInfo
	Meta        Meta

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Details *DocDetails `gorm:"foreignkey:DetailID;references:ID" json:"-"`
	Fields  []*Field    `gorm:"foreignkey:DocumentID;references:ID" json:"-"`
}

func (*Document) TableName() string {
	return "docs.documents"
}
