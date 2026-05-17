package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ismaelfi/auto-bidd/internal/middleware"
	"github.com/ismaelfi/auto-bidd/internal/models"
	"github.com/ismaelfi/auto-bidd/internal/services"
	"github.com/ismaelfi/auto-bidd/internal/views"
	"gorm.io/gorm"
)

type ChatHandler struct {
	db       *gorm.DB
	bids     *services.BidService
	renderer *views.Renderer
}

func NewChatHandler(db *gorm.DB, bids *services.BidService, renderer *views.Renderer) *ChatHandler {
	return &ChatHandler{db: db, bids: bids, renderer: renderer}
}

// SendMessage handles chat refinement requests
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	bidID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	if message == "" {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Message cannot be empty"})
		return
	}

	bid, err := h.bids.RefineBid(r.Context(), bidID, user.ID, message)
	if err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Refinement failed: " + err.Error()})
		return
	}

	// Return updated bid output + new chat messages
	var messages []models.ChatMessage
	h.db.Where("bid_id = ?", bidID).Order("created_at asc").Find(&messages)

	h.renderer.Partial(w, "bid_chat_response", map[string]any{
		"Bid":      bid,
		"Messages": messages,
	})
}

// GetMessages returns chat history
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	bidID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Verify ownership
	var bid models.Bid
	if err := h.db.Where("id = ? AND user_id = ?", bidID, user.ID).First(&bid).Error; err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var messages []models.ChatMessage
	h.db.Where("bid_id = ?", bidID).Order("created_at asc").Find(&messages)

	h.renderer.Partial(w, "chat_messages", map[string]any{
		"Messages": messages,
	})
}
