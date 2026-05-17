package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/ismaelfi/auto-bidd/internal/handlers"
	"github.com/ismaelfi/auto-bidd/internal/middleware"
	"github.com/ismaelfi/auto-bidd/internal/services"
	"github.com/ismaelfi/auto-bidd/internal/views"
	"gorm.io/gorm"
)

func New(db *gorm.DB, renderer *views.Renderer, aiService *services.AIService) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)

	// Static files
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Services
	authService := services.NewAuthService(db)
	bidService := services.NewBidService(db, aiService)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, renderer)
	dashHandler := handlers.NewDashboardHandler(renderer)
	profileHandler := handlers.NewProfileHandler(db, renderer)
	bidsHandler := handlers.NewBidsHandler(db, bidService, renderer)
	chatHandler := handlers.NewChatHandler(db, bidService, renderer)
	templatesHandler := handlers.NewTemplatesHandler(db, renderer)

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/login", authHandler.LoginPage)
		r.Get("/register", authHandler.RegisterPage)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/logout", authHandler.Logout)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(db))
		r.Get("/", dashHandler.Index)

		// Profile
		r.Get("/profile", profileHandler.Page)
		r.Put("/api/profile", profileHandler.UpdateProfile)
		r.Put("/api/profile/instructions", profileHandler.UpdateInstructions)
		r.Post("/api/profile/tone", profileHandler.AddTone)
		r.Delete("/api/profile/tone/{id}", profileHandler.DeleteTone)
		r.Post("/api/profile/portfolio", profileHandler.AddPortfolio)
		r.Delete("/api/profile/portfolio/{id}", profileHandler.DeletePortfolio)

		// Bids
		r.Get("/bids", bidsHandler.ListPage)
		r.Get("/bids/new", bidsHandler.NewPage)
		r.Get("/bids/{id}", bidsHandler.DetailPage)
		r.Post("/api/bids", bidsHandler.Create)
		r.Put("/api/bids/{id}", bidsHandler.Update)
		r.Patch("/api/bids/{id}/status", bidsHandler.UpdateStatus)
		r.Delete("/api/bids/{id}", bidsHandler.Delete)

		// Chat
		r.Post("/api/bids/{id}/chat", chatHandler.SendMessage)
		r.Get("/api/bids/{id}/chat", chatHandler.GetMessages)

		// Templates
		r.Get("/templates", templatesHandler.ListPage)
		r.Post("/api/bids/{id}/template", templatesHandler.CreateFromBid)
		r.Delete("/api/templates/{id}", templatesHandler.Delete)
	})

	return r
}
