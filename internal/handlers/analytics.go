package handlers

import (
	"net/http"
	"time"

	"github.com/ismaelfi/auto-bidd/internal/middleware"
	"github.com/ismaelfi/auto-bidd/internal/services"
	"github.com/ismaelfi/auto-bidd/internal/views"
)

type AnalyticsHandler struct {
	analytics *services.AnalyticsService
	renderer  *views.Renderer
}

func NewAnalyticsHandler(analytics *services.AnalyticsService, renderer *views.Renderer) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics, renderer: renderer}
}

func (h *AnalyticsHandler) Page(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	since := parsePeriod(r.URL.Query().Get("period"))

	overview := h.analytics.Overview(user.ID, since)
	brackets := h.analytics.ByPriceBracket(user.ID, since)
	trends := h.analytics.Trends(user.ID, since)
	templates := h.analytics.TemplateEffectiveness(user.ID, since)

	h.renderer.Page(w, "app", "analytics", map[string]any{
		"User":      user,
		"Overview":  overview,
		"Brackets":  brackets,
		"Trends":    trends,
		"Templates": templates,
		"Period":    r.URL.Query().Get("period"),
	})
}

func parsePeriod(p string) time.Time {
	now := time.Now()
	switch p {
	case "30d":
		return now.AddDate(0, 0, -30)
	case "6m":
		return now.AddDate(0, -6, 0)
	case "1y":
		return now.AddDate(-1, 0, 0)
	case "all":
		return time.Time{}
	default: // 90d
		return now.AddDate(0, 0, -90)
	}
}
