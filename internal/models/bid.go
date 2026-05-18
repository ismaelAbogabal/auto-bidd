package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BidStatus string

const (
	StatusDraft     BidStatus = "draft"
	StatusSubmitted BidStatus = "submitted"
	StatusWon       BidStatus = "won"
	StatusLost      BidStatus = "lost"
	StatusWithdrawn BidStatus = "withdrawn"
)

type QA struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Bid struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	TemplateID     *uuid.UUID `gorm:"type:uuid;index"`
	JobTitle       string     `gorm:"size:500;not null"`
	JobDescription string     `gorm:"type:text;not null"`
	Questions      string     `gorm:"type:text"`
	JobBudgetMin   float64    `gorm:"type:decimal(10,2)"`
	JobBudgetMax   float64    `gorm:"type:decimal(10,2)"`
	CoverLetter    string     `gorm:"type:text"`
	EstimatedHours int
	HourlyRate     float64   `gorm:"type:decimal(10,2)"`
	TotalPrice     float64   `gorm:"type:decimal(10,2)"`
	QAAnswers      []QA      `gorm:"type:jsonb;serializer:json"`
	Status         BidStatus `gorm:"type:varchar(20);default:'draft'"`
	Platform       string    `gorm:"size:100"`
	SubmittedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Messages []ChatMessage
	Template *Template `gorm:"foreignKey:TemplateID"`
}

func (b *Bid) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type ChatMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BidID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Role      string    `gorm:"type:varchar(20);not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time
}

func (c *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
