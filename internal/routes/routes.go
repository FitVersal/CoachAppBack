package routes

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	websocket "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saeid-a/CoachAppBack/internal/config"
	"github.com/saeid-a/CoachAppBack/internal/handlers"
	"github.com/saeid-a/CoachAppBack/internal/middleware"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/internal/services"
	chatws "github.com/saeid-a/CoachAppBack/internal/websocket"
	"github.com/saeid-a/CoachAppBack/pkg/utils"
)

func RegisterRoutes(app *fiber.App, cfg *config.Config, db *pgxpool.Pool) error {
	if err := registerDocsRoutes(app, cfg); err != nil {
		return err
	}

	userRepo := repository.NewUserRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	coachProfileRepo := repository.NewCoachProfileRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	programRepo := repository.NewWorkoutProgramRepository(db)
	conversationRepo := repository.NewConversationRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	var storageService services.StorageService
	if cfg.SupabaseURL != "" && cfg.SupabaseBucket != "" && cfg.SupabaseServiceKey != "" {
		storageService = services.NewSupabaseStorageService(
			cfg.SupabaseURL,
			cfg.SupabaseBucket,
			cfg.SupabaseServiceKey,
		)
	}
	if err := ensureDefaultUsers(cfg, db, userRepo, userProfileRepo, coachProfileRepo); err != nil {
		return err
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
	profileHandler := handlers.NewProfileHandler(
		profileService,
		userProfileRepo,
		coachProfileRepo,
		storageService,
	)
	matchmakingService := services.NewMatchmakingService(coachProfileRepo)
	coachDiscoveryHandler := handlers.NewCoachDiscoveryHandler(
		coachProfileRepo,
		userProfileRepo,
		matchmakingService,
	)
	sessionService := services.NewSessionService(
		db,
		sessionRepo,
		paymentRepo,
		userRepo,
		coachProfileRepo,
	)
	sessionHandler := handlers.NewSessionHandler(sessionService)
	programService := services.NewProgramService(
		db,
		programRepo,
		sessionRepo,
		userRepo,
		storageService,
	)
	programHandler := handlers.NewProgramHandler(programService)
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

	programs := authProtected.Group("/programs")
	programs.Post("", programHandler.CreateProgram)
	programs.Get("", programHandler.ListPrograms)
	programs.Get("/:id", programHandler.GetProgram)
	programs.Get("/:id/download", programHandler.DownloadProgram)

	conversations := authProtected.Group("/conversations")
	conversations.Get("", chatHandler.ListConversations)
	conversations.Post("", chatHandler.CreateConversation)
	conversations.Get("/:id/messages", chatHandler.GetMessages)

	api.Use("/v1/ws", chatHandler.WebSocketAuth)
	api.Get("/v1/ws", websocket.New(chatHandler.HandleWebSocket))

	return nil
}

func ensureDefaultUsers(
	cfg *config.Config,
	db *pgxpool.Pool,
	userRepo *repository.UserRepository,
	userProfileRepo *repository.UserProfileRepository,
	coachProfileRepo *repository.CoachProfileRepository,
) error {
	if cfg == nil {
		return nil
	}
	userEmail := strings.TrimSpace(cfg.DefaultUserEmail)
	userPassword := cfg.DefaultUserPassword
	userRole := strings.ToLower(strings.TrimSpace(cfg.DefaultUserRole))
	if userRole == "" {
		userRole = "user"
	}
	if userRole != "user" && userRole != "coach" {
		return fmt.Errorf("DEFAULT_USER_ROLE must be user or coach")
	}
	if err := ensureDefaultAccount(
		db,
		userRepo,
		userProfileRepo,
		coachProfileRepo,
		userEmail,
		userPassword,
		userRole,
	); err != nil {
		return err
	}
	if err := ensureDefaultAccount(
		db,
		userRepo,
		userProfileRepo,
		coachProfileRepo,
		strings.TrimSpace(cfg.DefaultCoachEmail),
		cfg.DefaultCoachPassword,
		"coach",
	); err != nil {
		return err
	}

	return nil
}

func ensureDefaultAccount(
	db *pgxpool.Pool,
	userRepo *repository.UserRepository,
	userProfileRepo *repository.UserProfileRepository,
	coachProfileRepo *repository.CoachProfileRepository,
	email string,
	password string,
	role string,
) error {
	if email == "" || password == "" {
		return nil
	}
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}
	email = strings.ToLower(parsedEmail.Address)
	if role != "user" && role != "coach" {
		return fmt.Errorf("role must be user or coach")
	}

	ctx := context.Background()
	existing, err := userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txUserRepo := repository.NewUserRepository(tx)
	txUserProfileRepo := repository.NewUserProfileRepository(tx)
	txCoachProfileRepo := repository.NewCoachProfileRepository(tx)

	user := &models.User{
		Email:        email,
		PasswordHash: hashed,
		Role:         role,
	}
	if err := txUserRepo.CreateUser(ctx, user); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil
		}
		return err
	}

	if role == "user" {
		if err := txUserProfileRepo.CreateEmpty(ctx, user.ID); err != nil {
			return err
		}
	} else {
		if err := txCoachProfileRepo.CreateEmpty(ctx, user.ID); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
