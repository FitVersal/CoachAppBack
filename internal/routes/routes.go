package routes

import (
	websocket "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saeid-a/CoachAppBack/internal/config"
	"github.com/saeid-a/CoachAppBack/internal/handlers"
	"github.com/saeid-a/CoachAppBack/internal/middleware"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
	chatws "github.com/saeid-a/CoachAppBack/internal/websocket"
)

func RegisterRoutes(app *fiber.App, cfg *config.Config, db *pgxpool.Pool) {
	userRepo := repository.NewUserRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	coachProfileRepo := repository.NewCoachProfileRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	conversationRepo := repository.NewConversationRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	var storageService services.StorageService
	if cfg.SupabaseURL != "" && cfg.SupabaseBucket != "" && cfg.SupabaseServiceKey != "" {
		storageService = services.NewSupabaseStorageService(cfg.SupabaseURL, cfg.SupabaseBucket, cfg.SupabaseServiceKey)
	}

	authHandler := handlers.NewAuthHandler(
		db,
		userRepo,
		userProfileRepo,
		coachProfileRepo,
		cfg.JWTSecret,
	)
	onboardingHandler := handlers.NewOnboardingHandler(userProfileRepo, coachProfileRepo)
	profileService := services.NewProfileService(userProfileRepo, coachProfileRepo)
	profileHandler := handlers.NewProfileHandler(profileService, userProfileRepo, coachProfileRepo, storageService)
	matchmakingService := services.NewMatchmakingService(coachProfileRepo)
	coachDiscoveryHandler := handlers.NewCoachDiscoveryHandler(coachProfileRepo, userProfileRepo, matchmakingService)
	sessionService := services.NewSessionService(db, sessionRepo, paymentRepo, userRepo, coachProfileRepo)
	sessionHandler := handlers.NewSessionHandler(sessionService)
	chatHub := chatws.NewHub()
	go chatHub.Run()
	chatService := services.NewChatService(db, conversationRepo, messageRepo, userRepo)
	chatHandler := handlers.NewChatHandler(chatService, chatHub, cfg.JWTSecret)

	api := app.Group("/api")

	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Get("/me", middleware.AuthRequired(cfg.JWTSecret), authHandler.Me)

	authProtected := api.Group("/v1", middleware.AuthRequired(cfg.JWTSecret))

	users := authProtected.Group("/users")
	users.Post("/onboarding", onboardingHandler.UserOnboarding)
	users.Get("/profile", profileHandler.GetUserProfile)
	users.Put("/profile", profileHandler.UpdateUserProfile)
	users.Post("/profile/avatar", profileHandler.UploadUserAvatar)

	coaches := authProtected.Group("/coaches")
	coaches.Get("", coachDiscoveryHandler.ListCoaches)
	coaches.Post("/onboarding", onboardingHandler.CoachOnboarding)
	coaches.Get("/profile", profileHandler.GetCoachProfile)
	coaches.Put("/profile", profileHandler.UpdateCoachProfile)
	coaches.Post("/profile/avatar", profileHandler.UploadCoachAvatar)
	coaches.Get("/recommended", coachDiscoveryHandler.GetRecommendedCoaches)
	coaches.Get("/:id", coachDiscoveryHandler.GetCoachDetail)

	sessions := authProtected.Group("/sessions")
	sessions.Post("/book", sessionHandler.BookSession)
	sessions.Get("", sessionHandler.ListSessions)
	sessions.Get("/:id", sessionHandler.GetSession)
	sessions.Put("/:id/status", sessionHandler.UpdateStatus)
	sessions.Post("/:id/pay", sessionHandler.PayForSession)

	conversations := authProtected.Group("/conversations")
	conversations.Get("", chatHandler.ListConversations)
	conversations.Post("", chatHandler.CreateConversation)
	conversations.Get("/:id/messages", chatHandler.GetMessages)

	api.Use("/v1/ws", chatHandler.WebSocketAuth)
	api.Get("/v1/ws", websocket.New(chatHandler.HandleWebSocket))
}
