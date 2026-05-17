package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserProfile struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	FullName       string    `gorm:"size:255"`
	Title          string    `gorm:"size:255"`
	HourlyRate     float64   `gorm:"type:decimal(10,2);default:0"`
	AIInstructions string    `gorm:"type:text"`
	Skills         []string  `gorm:"type:jsonb;serializer:json"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (p *UserProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type ToneExample struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Label     string    `gorm:"size:255"`
	Content   string    `gorm:"type:text;not null"`
	Context   string    `gorm:"type:text"`
	CreatedAt time.Time
}

func (t *ToneExample) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type PortfolioItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Title       string    `gorm:"size:255;not null"`
	Description string    `gorm:"type:text"`
	TechStack   []string  `gorm:"type:jsonb;serializer:json"`
	Outcome     string    `gorm:"type:text"`
	URL         string    `gorm:"size:500"`
	CreatedAt   time.Time
}

func (p *PortfolioItem) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
