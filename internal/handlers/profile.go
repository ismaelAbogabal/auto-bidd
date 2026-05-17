package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ismaelfi/auto-bidd/internal/middleware"
	"github.com/ismaelfi/auto-bidd/internal/models"
	"github.com/ismaelfi/auto-bidd/internal/views"
	"gorm.io/gorm"
)

type ProfileHandler struct {
	db       *gorm.DB
	renderer *views.Renderer
}

func NewProfileHandler(db *gorm.DB, renderer *views.Renderer) *ProfileHandler {
	return &ProfileHandler{db: db, renderer: renderer}
}

func (h *ProfileHandler) Page(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var profile models.UserProfile
	h.db.Where("user_id = ?", user.ID).First(&profile)

	var tones []models.ToneExample
	h.db.Where("user_id = ?", user.ID).Order("created_at desc").Find(&tones)

	var portfolio []models.PortfolioItem
	h.db.Where("user_id = ?", user.ID).Order("created_at desc").Find(&portfolio)

	h.renderer.Page(w, "app", "profile", map[string]any{
		"User":      user,
		"Profile":   profile,
		"Tones":     tones,
		"Portfolio": portfolio,
	})
}

// UpdateProfile handles basic info updates
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var profile models.UserProfile
	h.db.Where("user_id = ?", user.ID).First(&profile)

	profile.FullName = r.FormValue("full_name")
	profile.Title = r.FormValue("title")

	if rate, err := strconv.ParseFloat(r.FormValue("hourly_rate"), 64); err == nil {
		profile.HourlyRate = rate
	}

	// Parse skills from JSON hidden input
	if skillsJSON := r.FormValue("skills"); skillsJSON != "" {
		var skills []string
		if err := json.Unmarshal([]byte(skillsJSON), &skills); err == nil {
			profile.Skills = skills
		}
	} else {
		profile.Skills = []string{}
	}

	h.db.Save(&profile)

	h.renderer.Partial(w, "alert", map[string]any{"Type": "success", "Message": "Profile updated"})
}

// UpdateInstructions handles AI instructions update
func (h *ProfileHandler) UpdateInstructions(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var profile models.UserProfile
	h.db.Where("user_id = ?", user.ID).First(&profile)

	profile.AIInstructions = r.FormValue("ai_instructions")
	h.db.Save(&profile)

	h.renderer.Partial(w, "alert", map[string]any{"Type": "success", "Message": "AI instructions saved"})
}

// AddTone adds a new tone example
func (h *ProfileHandler) AddTone(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	content := r.FormValue("content")
	if len(content) < 20 {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Tone example must be at least 20 characters"})
		return
	}

	tone := &models.ToneExample{
		UserID:  user.ID,
		Label:   r.FormValue("label"),
		Content: content,
		Context: r.FormValue("context"),
	}

	h.db.Create(tone)

	h.renderer.Partial(w, "tone_item", tone)
}

// DeleteTone removes a tone example
func (h *ProfileHandler) DeleteTone(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	h.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.ToneExample{})
	w.WriteHeader(http.StatusOK)
}

// AddPortfolio adds a new portfolio item
func (h *ProfileHandler) AddPortfolio(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	title := r.FormValue("title")
	if title == "" {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Title is required"})
		return
	}

	var techStack []string
	if ts := r.FormValue("tech_stack"); ts != "" {
		json.Unmarshal([]byte(ts), &techStack)
	}

	item := &models.PortfolioItem{
		UserID:      user.ID,
		Title:       title,
		Description: r.FormValue("description"),
		TechStack:   techStack,
		Outcome:     r.FormValue("outcome"),
		URL:         r.FormValue("url"),
	}

	h.db.Create(item)

	h.renderer.Partial(w, "portfolio_item", item)
}

// DeletePortfolio removes a portfolio item
func (h *ProfileHandler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	h.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.PortfolioItem{})
	w.WriteHeader(http.StatusOK)
}
