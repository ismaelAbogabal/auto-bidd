package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ismaelfi/auto-bidd/internal/middleware"
	"github.com/ismaelfi/auto-bidd/internal/models"
	"github.com/ismaelfi/auto-bidd/internal/views"
	"gorm.io/gorm"
)

type TemplatesHandler struct {
	db       *gorm.DB
	renderer *views.Renderer
}

func NewTemplatesHandler(db *gorm.DB, renderer *views.Renderer) *TemplatesHandler {
	return &TemplatesHandler{db: db, renderer: renderer}
}

// ListPage shows all templates
func (h *TemplatesHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var templates []models.Template
	h.db.Where("user_id = ?", user.ID).Order("win_count desc").Find(&templates)

	h.renderer.Page(w, "app", "templates", map[string]any{
		"User":      user,
		"Templates": templates,
	})
}

// CreateFromBid saves a bid as a template
func (h *TemplatesHandler) CreateFromBid(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	bidID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var bid models.Bid
	if err := h.db.Where("id = ? AND user_id = ?", bidID, user.ID).First(&bid).Error; err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Bid not found"})
		return
	}

	// Determine initial win count
	winCount := 0
	if bid.Status == models.StatusWon {
		winCount = 1
	}

	tmpl := &models.Template{
		UserID:              user.ID,
		SourceBidID:         bid.ID,
		Name:                bid.JobTitle,
		CoverLetterTemplate: bid.CoverLetter,
		Tags:                []string{},
		WinCount:            winCount,
		UseCount:            0,
	}

	if err := h.db.Create(tmpl).Error; err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Failed to create template"})
		return
	}

	h.renderer.Partial(w, "alert", map[string]any{"Type": "success", "Message": "Template saved: " + tmpl.Name})
}

// Delete removes a template
func (h *TemplatesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	h.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Template{})
	w.WriteHeader(http.StatusOK)
}
