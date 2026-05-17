package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/ismaelfi/auto-bidd/internal/models"
	"gorm.io/gorm"
)

type BidService struct {
	db *gorm.DB
	ai *AIService
}

func NewBidService(db *gorm.DB, ai *AIService) *BidService {
	return &BidService{db: db, ai: ai}
}

type CreateBidInput struct {
	JobTitle       string
	JobDescription string
	Questions      string // newline-separated
	BudgetMin      float64
	BudgetMax      float64
	Platform       string
	TemplateID     *uuid.UUID
}

func (s *BidService) GenerateBid(ctx context.Context, userID uuid.UUID, input CreateBidInput) (*models.Bid, error) {
	// Load user profile data
	var profile models.UserProfile
	s.db.Where("user_id = ?", userID).First(&profile)

	var tones []models.ToneExample
	s.db.Where("user_id = ?", userID).Find(&tones)

	var portfolio []models.PortfolioItem
	s.db.Where("user_id = ?", userID).Find(&portfolio)

	// Filter relevant portfolio items
	relevantPortfolio := selectRelevantPortfolio(portfolio, input.JobDescription)

	// Build prompts
	systemPrompt := s.ai.BuildSystemPrompt(&profile, tones, relevantPortfolio)

	// Parse questions
	var questions []string
	for _, q := range strings.Split(input.Questions, "\n") {
		q = strings.TrimSpace(q)
		if q != "" {
			questions = append(questions, q)
		}
	}

	// Load template cover letter if specified
	var templateCover string
	if input.TemplateID != nil {
		var tmpl models.Template
		if err := s.db.First(&tmpl, "id = ?", *input.TemplateID).Error; err == nil {
			templateCover = tmpl.CoverLetterTemplate
			// Increment use count
			s.db.Model(&tmpl).Update("use_count", gorm.Expr("use_count + 1"))
		}
	}

	aiReq := GenerateBidRequest{
		JobTitle:       input.JobTitle,
		JobDescription: input.JobDescription,
		Questions:      questions,
		BudgetMin:      input.BudgetMin,
		BudgetMax:      input.BudgetMax,
		TemplateCover:  templateCover,
	}

	userMessage := s.ai.BuildUserMessage(aiReq)

	// Call AI
	aiResp, err := s.ai.Generate(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, err
	}

	// Calculate pricing
	hourlyRate := profile.HourlyRate
	if hourlyRate == 0 {
		hourlyRate = 50 // default fallback
	}
	totalPrice := float64(aiResp.EstimatedHours) * hourlyRate

	// Create bid record
	bid := &models.Bid{
		UserID:         userID,
		TemplateID:     input.TemplateID,
		JobTitle:       input.JobTitle,
		JobDescription: input.JobDescription,
		JobBudgetMin:   input.BudgetMin,
		JobBudgetMax:   input.BudgetMax,
		CoverLetter:    aiResp.CoverLetter,
		EstimatedHours: aiResp.EstimatedHours,
		HourlyRate:     hourlyRate,
		TotalPrice:     totalPrice,
		QAAnswers:      aiResp.QAAnswers,
		Status:         models.StatusDraft,
		Platform:       input.Platform,
	}

	if err := s.db.Create(bid).Error; err != nil {
		return nil, err
	}

	// Save AI response as first chat message
	s.db.Create(&models.ChatMessage{
		BidID:   bid.ID,
		Role:    "assistant",
		Content: formatBidAsJSON(aiResp),
	})

	return bid, nil
}

func (s *BidService) RefineBid(ctx context.Context, bidID uuid.UUID, userID uuid.UUID, message string) (*models.Bid, error) {
	var bid models.Bid
	if err := s.db.Where("id = ? AND user_id = ?", bidID, userID).First(&bid).Error; err != nil {
		return nil, err
	}

	// Load profile for system prompt
	var profile models.UserProfile
	s.db.Where("user_id = ?", userID).First(&profile)

	var tones []models.ToneExample
	s.db.Where("user_id = ?", userID).Find(&tones)

	var portfolio []models.PortfolioItem
	s.db.Where("user_id = ?", userID).Find(&portfolio)

	relevantPortfolio := selectRelevantPortfolio(portfolio, bid.JobDescription)
	systemPrompt := s.ai.BuildSystemPrompt(&profile, tones, relevantPortfolio)

	// Load chat history
	var history []models.ChatMessage
	s.db.Where("bid_id = ?", bidID).Order("created_at asc").Find(&history)

	// Save user message
	s.db.Create(&models.ChatMessage{
		BidID:   bidID,
		Role:    "user",
		Content: message,
	})

	// Call AI with history
	aiResp, err := s.ai.Refine(ctx, systemPrompt, history, message)
	if err != nil {
		return nil, err
	}

	// Update bid
	hourlyRate := bid.HourlyRate
	bid.CoverLetter = aiResp.CoverLetter
	bid.EstimatedHours = aiResp.EstimatedHours
	bid.TotalPrice = float64(aiResp.EstimatedHours) * hourlyRate
	if aiResp.QAAnswers != nil {
		bid.QAAnswers = aiResp.QAAnswers
	}

	s.db.Save(&bid)

	// Save assistant response
	s.db.Create(&models.ChatMessage{
		BidID:   bidID,
		Role:    "assistant",
		Content: formatBidAsJSON(aiResp),
	})

	return &bid, nil
}

func (s *BidService) UpdateBid(bid *models.Bid) error {
	bid.TotalPrice = float64(bid.EstimatedHours) * bid.HourlyRate
	return s.db.Save(bid).Error
}

func (s *BidService) UpdateStatus(bidID, userID uuid.UUID, status models.BidStatus) error {
	result := s.db.Model(&models.Bid{}).
		Where("id = ? AND user_id = ?", bidID, userID).
		Update("status", status)

	if status == models.StatusWon {
		// Increment template win count if bid used a template
		var bid models.Bid
		if err := s.db.First(&bid, "id = ?", bidID).Error; err == nil && bid.TemplateID != nil {
			s.db.Model(&models.Template{}).Where("id = ?", bid.TemplateID).
				Update("win_count", gorm.Expr("win_count + 1"))
		}
	}

	return result.Error
}

// selectRelevantPortfolio picks up to 3 items with tech stack overlap
func selectRelevantPortfolio(items []models.PortfolioItem, jobDesc string) []models.PortfolioItem {
	if len(items) <= 3 {
		return items
	}

	jobLower := strings.ToLower(jobDesc)

	type scored struct {
		item  models.PortfolioItem
		score int
	}

	var results []scored
	for _, item := range items {
		score := 0
		for _, tech := range item.TechStack {
			if strings.Contains(jobLower, strings.ToLower(tech)) {
				score++
			}
		}
		// Also check title/description keywords
		if strings.Contains(jobLower, strings.ToLower(item.Title)) {
			score += 2
		}
		results = append(results, scored{item: item, score: score})
	}

	// Sort by score descending (simple bubble sort for small slice)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	out := make([]models.PortfolioItem, 0, 3)
	for i := 0; i < 3 && i < len(results); i++ {
		out = append(out, results[i].item)
	}
	return out
}

func formatBidAsJSON(resp *GenerateBidResponse) string {
	b, _ := json.Marshal(resp)
	return string(b)
}
