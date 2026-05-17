package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string    `gorm:"uniqueIndex;not null;size:255"`
	PasswordHash string    `gorm:"not null;size:255"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	Profile      UserProfile
	ToneExamples []ToneExample
	Portfolio    []PortfolioItem
	Bids         []Bid
	Templates    []Template
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Token     string    `gorm:"uniqueIndex;not null;size:64"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
