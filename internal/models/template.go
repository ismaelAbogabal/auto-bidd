package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Template struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID              uuid.UUID `gorm:"type:uuid;not null;index"`
	SourceBidID         uuid.UUID `gorm:"type:uuid"`
	Name                string    `gorm:"size:255;not null"`
	CoverLetterTemplate string    `gorm:"type:text"`
	Tags                []string  `gorm:"type:jsonb;serializer:json"`
	WinCount            int       `gorm:"default:0"`
	UseCount            int       `gorm:"default:0"`
	CreatedAt           time.Time
}

func (t *Template) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
