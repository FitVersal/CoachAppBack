package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

var ErrStorageUnavailable = errors.New("storage service is not configured")

type workoutProgramStore interface {
	Create(
		ctx context.Context,
		input repository.CreateWorkoutProgramInput,
	) (*models.WorkoutProgram, error)
	ListByCoachID(ctx context.Context, coachID int64) ([]models.WorkoutProgram, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.WorkoutProgram, error)
	GetByID(ctx context.Context, programID int64) (*models.WorkoutProgram, error)
}

type ProgramService struct {
	programRepo    workoutProgramStore
	sessionRepo    *repository.SessionRepository
	userRepo       userReader
	storageService StorageService
}

type CreateProgramInput struct {
	UserID      int64
	SessionID   int64
	Title       string
	Description *string
	File        multipart.File
	Filename    string
}

func NewProgramService(
	db *pgxpool.Pool,
	programRepo *repository.WorkoutProgramRepository,
	sessionRepo *repository.SessionRepository,
	userRepo userReader,
	storageService StorageService,
) *ProgramService {
	return &ProgramService{
		programRepo:    programRepo,
		sessionRepo:    sessionRepo,
		userRepo:       userRepo,
		storageService: storageService,
	}
}

func (s *ProgramService) CreateProgram(
	ctx context.Context,
	coachID int64,
	input CreateProgramInput,
) (*models.WorkoutProgram, error) {
	if s.storageService == nil {
		return nil, ErrStorageUnavailable
	}
	if coachID <= 0 || input.UserID <= 0 || input.SessionID <= 0 || input.File == nil {
		return nil, ErrInvalidInput
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}

	var description *string
	if input.Description != nil {
		trimmed := strings.TrimSpace(*input.Description)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		description = &trimmed
	}

	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	if user.Role != "user" {
		return nil, ErrInvalidInput
	}

	session, err := s.sessionRepo.GetByID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}
	if session.CoachID != coachID || session.UserID != input.UserID {
		return nil, ErrForbidden
	}

	filename := buildProgramFilename(coachID, input.UserID, input.Filename)
	fileURL, err := s.storageService.UploadFile(ctx, input.File, filename, "programs")
	if err != nil {
		return nil, err
	}

	program, err := s.programRepo.Create(ctx, repository.CreateWorkoutProgramInput{
		CoachID:     coachID,
		UserID:      input.UserID,
		SessionID:   input.SessionID,
		Title:       title,
		Description: description,
		FileURL:     fileURL,
	})
	if err != nil {
		cleanupErr := s.storageService.DeleteFile(ctx, fileURL)
		if cleanupErr != nil {
			return nil, errors.Join(err, fmt.Errorf("cleanup failed: %w", cleanupErr))
		}
		return nil, err
	}

	return program, nil
}

func (s *ProgramService) ListPrograms(
	ctx context.Context,
	actorID int64,
	role string,
) ([]models.WorkoutProgram, error) {
	switch role {
	case "coach":
		return s.programRepo.ListByCoachID(ctx, actorID)
	case "user":
		return s.programRepo.ListByUserID(ctx, actorID)
	default:
		return nil, ErrForbidden
	}
}

func (s *ProgramService) GetProgram(
	ctx context.Context,
	actorID int64,
	role string,
	programID int64,
) (*models.WorkoutProgram, error) {
	program, err := s.programRepo.GetByID(ctx, programID)
	if err != nil {
		return nil, err
	}
	if !canAccessProgram(role, actorID, program) {
		return nil, ErrForbidden
	}
	return program, nil
}

func (s *ProgramService) GetDownloadURL(
	ctx context.Context,
	actorID int64,
	role string,
	programID int64,
) (string, error) {
	if s.storageService == nil {
		return "", ErrStorageUnavailable
	}

	program, err := s.GetProgram(ctx, actorID, role, programID)
	if err != nil {
		return "", err
	}

	return s.storageService.GetSignedURL(ctx, program.FileURL)
}

func canAccessProgram(role string, actorID int64, program *models.WorkoutProgram) bool {
	if program == nil {
		return false
	}

	switch role {
	case "coach":
		return actorID == program.CoachID
	case "user":
		return actorID == program.UserID
	default:
		return false
	}
}

func buildProgramFilename(coachID int64, userID int64, original string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(original)))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%d-%d-%d%s", coachID, userID, time.Now().UnixNano(), ext)
}
