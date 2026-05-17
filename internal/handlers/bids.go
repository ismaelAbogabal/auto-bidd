package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ismaelfi/auto-bidd/internal/middleware"
	"github.com/ismaelfi/auto-bidd/internal/models"
	"github.com/ismaelfi/auto-bidd/internal/services"
	"github.com/ismaelfi/auto-bidd/internal/views"
	"gorm.io/gorm"
)

type BidsHandler struct {
	db       *gorm.DB
	bids     *services.BidService
	renderer *views.Renderer
}

func NewBidsHandler(db *gorm.DB, bids *services.BidService, renderer *views.Renderer) *BidsHandler {
	return &BidsHandler{db: db, bids: bids, renderer: renderer}
}

// ListPage renders the bid list
func (h *BidsHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var bids []models.Bid
	h.db.Where("user_id = ?", user.ID).Order("created_at desc").Find(&bids)

	h.renderer.Page(w, "app", "bid_list", map[string]any{
		"User": user,
		"Bids": bids,
	})
}

// NewPage renders the bid builder form
func (h *BidsHandler) NewPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var templates []models.Template
	h.db.Where("user_id = ?", user.ID).Order("win_count desc").Find(&templates)

	h.renderer.Page(w, "app", "bid_builder", map[string]any{
		"User":      user,
		"Templates": templates,
	})
}

// DetailPage renders a single bid
func (h *BidsHandler) DetailPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var bid models.Bid
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&bid).Error; err != nil {
		http.Error(w, "Bid not found", http.StatusNotFound)
		return
	}

	var messages []models.ChatMessage
	h.db.Where("bid_id = ?", bid.ID).Order("created_at asc").Find(&messages)

	h.renderer.Page(w, "app", "bid_detail", map[string]any{
		"User":     user,
		"Bid":      bid,
		"Messages": messages,
	})
}

// Create creates a bid record and returns HTML with the streaming component
func (h *BidsHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	jobTitle := r.FormValue("job_title")
	jobDesc := r.FormValue("job_description")

	if jobTitle == "" || jobDesc == "" {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Job title and description are required"})
		return
	}

	budgetMin, _ := strconv.ParseFloat(r.FormValue("budget_min"), 64)
	budgetMax, _ := strconv.ParseFloat(r.FormValue("budget_max"), 64)

	var templateID *uuid.UUID
	if tid := r.FormValue("template_id"); tid != "" {
		if parsed, err := uuid.Parse(tid); err == nil {
			templateID = &parsed
		}
	}

	input := services.CreateBidInput{
		JobTitle:       jobTitle,
		JobDescription: jobDesc,
		Questions:      r.FormValue("questions"),
		BudgetMin:      budgetMin,
		BudgetMax:      budgetMax,
		Platform:       r.FormValue("platform"),
		TemplateID:     templateID,
	}

	bid, err := h.bids.CreateBidRecord(user.ID, input)
	if err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Failed to create bid"})
		return
	}

	// Return the streaming component
	h.renderer.Partial(w, "bid_stream", map[string]any{
		"BidID":    bid.ID.String(),
		"Endpoint": fmt.Sprintf("/api/bids/%s/generate", bid.ID),
	})
}

// StreamGenerate is the SSE endpoint for bid generation
func (h *BidsHandler) StreamGenerate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var bid models.Bid
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&bid).Error; err != nil {
		http.Error(w, "Bid not found", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	err = h.bids.StreamGenerate(r.Context(), &bid, func(text string) {
		fmt.Fprintf(w, "event: delta\ndata: %s\n\n", escapeSSE(text))
		flusher.Flush()
	})

	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", escapeSSE(err.Error()))
		flusher.Flush()
		return
	}

	fmt.Fprintf(w, "event: done\ndata: {\"redirect\":\"/bids/%s\"}\n\n", bid.ID)
	flusher.Flush()
}

// StreamRefine is the SSE endpoint for chat refinement
func (h *BidsHandler) StreamRefine(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var bid models.Bid
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&bid).Error; err != nil {
		http.Error(w, "Bid not found", http.StatusNotFound)
		return
	}

	message := r.URL.Query().Get("message")
	if message == "" {
		http.Error(w, "Message required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	err = h.bids.StreamRefine(r.Context(), &bid, message, func(text string) {
		fmt.Fprintf(w, "event: delta\ndata: %s\n\n", escapeSSE(text))
		flusher.Flush()
	})

	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", escapeSSE(err.Error()))
		flusher.Flush()
		return
	}

	fmt.Fprintf(w, "event: done\ndata: {\"redirect\":\"/bids/%s\"}\n\n", bid.ID)
	flusher.Flush()
}

// Update handles manual edits to a bid
func (h *BidsHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var bid models.Bid
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&bid).Error; err != nil {
		http.Error(w, "Bid not found", http.StatusNotFound)
		return
	}

	if cl := r.FormValue("cover_letter"); cl != "" {
		bid.CoverLetter = cl
	}
	if hours, err := strconv.Atoi(r.FormValue("estimated_hours")); err == nil && hours > 0 {
		bid.EstimatedHours = hours
	}
	if rate, err := strconv.ParseFloat(r.FormValue("hourly_rate"), 64); err == nil && rate > 0 {
		bid.HourlyRate = rate
	}

	h.bids.UpdateBid(&bid)

	h.renderer.Partial(w, "bid_output", map[string]any{"Bid": bid})
}

// UpdateStatus changes bid status
func (h *BidsHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	status := models.BidStatus(r.FormValue("status"))
	if err := h.bids.UpdateStatus(id, user.ID, status); err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": err.Error()})
		return
	}

	if status == models.StatusSubmitted {
		now := time.Now()
		h.db.Model(&models.Bid{}).Where("id = ?", id).Update("submitted_at", &now)
	}

	h.renderer.Partial(w, "alert", map[string]any{"Type": "success", "Message": "Status updated to " + string(status)})
}

// Delete removes a bid
func (h *BidsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	h.db.Where("bid_id = ?", id).Delete(&models.ChatMessage{})
	h.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Bid{})

	w.Header().Set("HX-Redirect", "/bids")
	w.WriteHeader(http.StatusOK)
}

// escapeSSE escapes newlines for SSE data field
func escapeSSE(s string) string {
	s = fmt.Sprintf("%q", s)
	// Remove surrounding quotes from %q
	return s[1 : len(s)-1]
}
