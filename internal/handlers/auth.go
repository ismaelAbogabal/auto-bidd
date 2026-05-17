package handlers

import (
	"net/http"
	"time"

	"github.com/ismaelfi/auto-bidd/internal/services"
	"github.com/ismaelfi/auto-bidd/internal/views"
)

type AuthHandler struct {
	auth     *services.AuthService
	renderer *views.Renderer
}

func NewAuthHandler(auth *services.AuthService, renderer *views.Renderer) *AuthHandler {
	return &AuthHandler{auth: auth, renderer: renderer}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.renderer.Page(w, "auth", "login", nil)
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	h.renderer.Page(w, "auth", "register", nil)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if email == "" || password == "" {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Email and password are required"})
		return
	}

	if len(password) < 8 {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Password must be at least 8 characters"})
		return
	}

	if password != confirm {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Passwords do not match"})
		return
	}

	user, err := h.auth.Register(email, password)
	if err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": err.Error()})
		return
	}

	token, err := h.auth.CreateSession(user.ID)
	if err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Failed to create session"})
		return
	}

	setSessionCookie(w, token)

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := h.auth.Login(email, password)
	if err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Invalid email or password"})
		return
	}

	token, err := h.auth.CreateSession(user.ID)
	if err != nil {
		h.renderer.Partial(w, "alert", map[string]any{"Type": "error", "Message": "Failed to create session"})
		return
	}

	setSessionCookie(w, token)

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.auth.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
