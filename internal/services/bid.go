package services

import (
	"context"
	"encoding/json"
	"log"
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

// CreateBidRecord creates a bid record with empty cover letter, ready for streaming generation.
func (s *BidService) CreateBidRecord(userID uuid.UUID, input CreateBidInput) (*models.Bid, error) {
	var profile models.UserProfile
	s.db.Where("user_id = ?", userID).First(&profile)

	hourlyRate := profile.HourlyRate
	if hourlyRate == 0 {
		hourlyRate = 50
	}

	// Increment template use count
	if input.TemplateID != nil {
		s.db.Model(&models.Template{}).Where("id = ?", *input.TemplateID).
			Update("use_count", gorm.Expr("use_count + 1"))
	}

	bid := &models.Bid{
		UserID:         userID,
		TemplateID:     input.TemplateID,
		JobTitle:       input.JobTitle,
		JobDescription: input.JobDescription,
		Questions:      input.Questions,
		JobBudgetMin:   input.BudgetMin,
		JobBudgetMax:   input.BudgetMax,
		HourlyRate:     hourlyRate,
		Status:         models.StatusDraft,
		Platform:       input.Platform,
	}

	if err := s.db.Create(bid).Error; err != nil {
		return nil, err
	}

	return bid, nil
}

// StreamGenerate runs streaming AI generation for a bid, calling onText for each chunk.
// Updates the bid record when complete.
func (s *BidService) StreamGenerate(ctx context.Context, bid *models.Bid, onText func(string)) error {
	log.Printf("[AI] loading user context for %s", bid.UserID)
	profile, tones, portfolio := s.loadUserContext(bid.UserID)
	relevantPortfolio := selectRelevantPortfolio(portfolio, bid.JobDescription)
	log.Printf("[AI] user context: profile=%q, tones=%d, portfolio=%d (relevant=%d)",
		profile.FullName, len(tones), len(portfolio), len(relevantPortfolio))

	systemPrompt := s.ai.BuildSystemPrompt(&profile, tones, relevantPortfolio)
	log.Printf("[AI] system prompt: %d chars", len(systemPrompt))

	questions := parseQuestions(bid.Questions)

	var templateCover string
	if bid.TemplateID != nil {
		var tmpl models.Template
		if err := s.db.First(&tmpl, "id = ?", *bid.TemplateID).Error; err == nil {
			templateCover = tmpl.CoverLetterTemplate
		}
	}

	userMessage := s.ai.BuildUserMessage(GenerateBidRequest{
		JobTitle:       bid.JobTitle,
		JobDescription: bid.JobDescription,
		Questions:      questions,
		BudgetMin:      bid.JobBudgetMin,
		BudgetMax:      bid.JobBudgetMax,
		TemplateCover:  templateCover,
	})
	log.Printf("[AI] user message: %d chars, questions=%d", len(userMessage), len(questions))

	log.Printf("[AI] calling LLM provider...")
	fullText, err := s.ai.GenerateStream(ctx, systemPrompt, userMessage, onText)
	if err != nil {
		log.Printf("[AI] stream error: %v", err)
		return err
	}

	log.Printf("[AI] stream finished, response: %d chars", len(fullText))
	if err := s.saveBidFromResponse(bid, fullText); err != nil {
		log.Printf("[AI] save error: %v", err)
		return err
	}

	// Save the initial exchange as chat history so refinement has context
	s.db.Create(&models.ChatMessage{
		BidID:   bid.ID,
		Role:    "user",
		Content: userMessage,
	})
	s.db.Create(&models.ChatMessage{
		BidID:   bid.ID,
		Role:    "assistant",
		Content: fullText,
	})

	log.Printf("[AI] bid %s saved successfully", bid.ID)
	return nil
}

// StreamRefine runs streaming AI refinement for a bid
func (s *BidService) StreamRefine(ctx context.Context, bid *models.Bid, message string, onText func(string)) error {
	profile, tones, portfolio := s.loadUserContext(bid.UserID)
	relevantPortfolio := selectRelevantPortfolio(portfolio, bid.JobDescription)
	systemPrompt := s.ai.BuildSystemPrompt(&profile, tones, relevantPortfolio)

	var history []models.ChatMessage
	s.db.Where("bid_id = ?", bid.ID).Order("created_at asc").Find(&history)

	// Save user message
	s.db.Create(&models.ChatMessage{
		BidID:   bid.ID,
		Role:    "user",
		Content: message,
	})

	fullText, err := s.ai.RefineStream(ctx, systemPrompt, history, message, onText)
	if err != nil {
		return err
	}

	// Save assistant response
	s.db.Create(&models.ChatMessage{
		BidID:   bid.ID,
		Role:    "assistant",
		Content: fullText,
	})

	return s.saveBidFromResponse(bid, fullText)
}

// GenerateBid is the non-streaming version (kept for compatibility)
func (s *BidService) GenerateBid(ctx context.Context, userID uuid.UUID, input CreateBidInput) (*models.Bid, error) {
	bid, err := s.CreateBidRecord(userID, input)
	if err != nil {
		return nil, err
	}

	profile, tones, portfolio := s.loadUserContext(userID)
	relevantPortfolio := selectRelevantPortfolio(portfolio, input.JobDescription)
	systemPrompt := s.ai.BuildSystemPrompt(&profile, tones, relevantPortfolio)

	questions := parseQuestions(input.Questions)

	var templateCover string
	if input.TemplateID != nil {
		var tmpl models.Template
		if err := s.db.First(&tmpl, "id = ?", *input.TemplateID).Error; err == nil {
			templateCover = tmpl.CoverLetterTemplate
		}
	}

	userMessage := s.ai.BuildUserMessage(GenerateBidRequest{
		JobTitle:       input.JobTitle,
		JobDescription: input.JobDescription,
		Questions:      questions,
		BudgetMin:      input.BudgetMin,
		BudgetMax:      input.BudgetMax,
		TemplateCover:  templateCover,
	})

	aiResp, err := s.ai.Generate(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, err
	}

	bid.CoverLetter = aiResp.CoverLetter
	bid.EstimatedHours = aiResp.EstimatedHours
	bid.TotalPrice = float64(aiResp.EstimatedHours) * bid.HourlyRate
	bid.QAAnswers = aiResp.QAAnswers
	s.db.Save(bid)

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

	profile, tones, portfolio := s.loadUserContext(userID)
	relevantPortfolio := selectRelevantPortfolio(portfolio, bid.JobDescription)
	systemPrompt := s.ai.BuildSystemPrompt(&profile, tones, relevantPortfolio)

	var history []models.ChatMessage
	s.db.Where("bid_id = ?", bidID).Order("created_at asc").Find(&history)

	s.db.Create(&models.ChatMessage{
		BidID:   bidID,
		Role:    "user",
		Content: message,
	})

	aiResp, err := s.ai.Refine(ctx, systemPrompt, history, message)
	if err != nil {
		return nil, err
	}

	bid.CoverLetter = aiResp.CoverLetter
	bid.EstimatedHours = aiResp.EstimatedHours
	bid.TotalPrice = float64(aiResp.EstimatedHours) * bid.HourlyRate
	if aiResp.QAAnswers != nil {
		bid.QAAnswers = aiResp.QAAnswers
	}
	s.db.Save(&bid)

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
		var bid models.Bid
		if err := s.db.First(&bid, "id = ?", bidID).Error; err == nil && bid.TemplateID != nil {
			s.db.Model(&models.Template{}).Where("id = ?", bid.TemplateID).
				Update("win_count", gorm.Expr("win_count + 1"))
		}
	}

	return result.Error
}

func (s *BidService) saveBidFromResponse(bid *models.Bid, fullText string) error {
	aiResp, err := ParseResponse(fullText)
	if err != nil {
		// Even if parsing fails, save the raw text as cover letter
		bid.CoverLetter = strings.TrimSpace(fullText)
		return s.db.Save(bid).Error
	}

	bid.CoverLetter = aiResp.CoverLetter
	bid.EstimatedHours = aiResp.EstimatedHours
	bid.TotalPrice = float64(aiResp.EstimatedHours) * bid.HourlyRate
	if aiResp.QAAnswers != nil {
		bid.QAAnswers = aiResp.QAAnswers
	}
	return s.db.Save(bid).Error
}

func (s *BidService) loadUserContext(userID uuid.UUID) (models.UserProfile, []models.ToneExample, []models.PortfolioItem) {
	var profile models.UserProfile
	s.db.Where("user_id = ?", userID).First(&profile)

	var tones []models.ToneExample
	s.db.Where("user_id = ?", userID).Find(&tones)

	var portfolio []models.PortfolioItem
	s.db.Where("user_id = ?", userID).Find(&portfolio)

	return profile, tones, portfolio
}

func parseQuestions(text string) []string {
	var questions []string
	for _, q := range strings.Split(text, "\n") {
		q = strings.TrimSpace(q)
		if q != "" {
			questions = append(questions, q)
		}
	}
	return questions
}

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
		if strings.Contains(jobLower, strings.ToLower(item.Title)) {
			score += 2
		}
		results = append(results, scored{item: item, score: score})
	}

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
