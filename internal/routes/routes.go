package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saeid-a/CoachAppBack/internal/config"
	"github.com/saeid-a/CoachAppBack/internal/handlers"
	"github.com/saeid-a/CoachAppBack/internal/middleware"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
)

func RegisterRoutes(app *fiber.App, cfg *config.Config, db *pgxpool.Pool) {
	userRepo := repository.NewUserRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	coachProfileRepo := repository.NewCoachProfileRepository(db)
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
	coaches.Post("/onboarding", onboardingHandler.CoachOnboarding)
	coaches.Get("/profile", profileHandler.GetCoachProfile)
	coaches.Put("/profile", profileHandler.UpdateCoachProfile)
	coaches.Post("/profile/avatar", profileHandler.UploadCoachAvatar)
}
