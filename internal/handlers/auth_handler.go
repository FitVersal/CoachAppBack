package handlers

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
	"github.com/saeid-a/CoachAppBack/pkg/utils"
)

type AuthHandler struct {
	db               *pgxpool.Pool
	userRepo         *repository.UserRepository
	userProfileRepo  *repository.UserProfileRepository
	coachProfileRepo *repository.CoachProfileRepository
	jwtSecret        string
}

func NewAuthHandler(
	db *pgxpool.Pool,
	userRepo *repository.UserRepository,
	userProfileRepo *repository.UserProfileRepository,
	coachProfileRepo *repository.CoachProfileRepository,
	jwtSecret string,
) *AuthHandler {
	return &AuthHandler{
		db:               db,
		userRepo:         userRepo,
		userProfileRepo:  userProfileRepo,
		coachProfileRepo: coachProfileRepo,
		jwtSecret:        jwtSecret,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	parsedEmail, err := mail.ParseAddress(strings.TrimSpace(req.Email))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid email format"})
	}
	req.Email = strings.ToLower(parsedEmail.Address)
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "Password must be at least 8 characters"})
	}
	if req.Role != "user" && req.Role != "coach" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role"})
	}

	existing, err := h.userRepo.GetByEmail(c.Context(), req.Email)
	if err == nil && existing != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email already exists"})
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to check email"})
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to hash password"})
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: hashed,
		Role:         req.Role,
	}
	tx, err := h.db.Begin(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to start registration transaction"})
	}
	defer func() {
		_ = tx.Rollback(c.Context())
	}()

	txUserRepo := repository.NewUserRepository(tx)
	txUserProfileRepo := repository.NewUserProfileRepository(tx)
	txCoachProfileRepo := repository.NewCoachProfileRepository(tx)

	if err := txUserRepo.CreateUser(c.Context(), user); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.Status(fiber.StatusConflict).
				JSON(fiber.Map{"error": "Email already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to create user"})
	}

	if req.Role == "user" {
		if err := txUserProfileRepo.CreateEmpty(c.Context(), user.ID); err != nil {
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "Failed to create user profile"})
		}
	} else {
		if err := txCoachProfileRepo.CreateEmpty(c.Context(), user.ID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create coach profile"})
		}
	}

	if err := tx.Commit(c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to finalize registration"})
	}

	token, err := utils.GenerateToken(strconv.FormatInt(user.ID, 10), user.Role, h.jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	parsedEmail, err := mail.ParseAddress(strings.TrimSpace(req.Email))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid email format"})
	}
	req.Email = strings.ToLower(parsedEmail.Address)

	user, err := h.userRepo.GetByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusUnauthorized).
				JSON(fiber.Map{"error": "Invalid email or password"})
		}
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to lookup user"})
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "Invalid email or password"})
	}

	token, err := utils.GenerateToken(strconv.FormatInt(user.ID, 10), user.Role, h.jwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userIDValue := c.Locals("user_id")
	roleValue := c.Locals("role")
	userIDStr, ok := userIDValue.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}
	role, ok := roleValue.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
	}

	if role == "user" {
		profile, err := h.userProfileRepo.GetByUserID(c.Context(), userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile"})
		}
		return c.JSON(fiber.Map{
			"user": fiber.Map{
				"id":    user.ID,
				"email": user.Email,
				"role":  user.Role,
			},
			"profile":             profile,
			"onboarding_complete": profile.OnboardingComplete,
		})
	}

	profile, err := h.coachProfileRepo.GetByUserID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profile not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile"})
	}
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
		"profile":             profile,
		"onboarding_complete": profile.OnboardingComplete,
	})
}
