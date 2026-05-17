package handlers

import (
	"net/http"

	"github.com/ismaelfi/auto-bidd/internal/middleware"
	"github.com/ismaelfi/auto-bidd/internal/views"
)

type DashboardHandler struct {
	renderer *views.Renderer
}

func NewDashboardHandler(renderer *views.Renderer) *DashboardHandler {
	return &DashboardHandler{renderer: renderer}
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	h.renderer.Page(w, "app", "dashboard", map[string]any{
		"User": user,
	})
}
