package services

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

type stubProgramRepo struct {
	createResult *models.WorkoutProgram
	createErr    error
	listResult   []models.WorkoutProgram
	listErr      error
	getResult    *models.WorkoutProgram
	getErr       error
	lastCreate   repository.CreateWorkoutProgramInput
}

func (r *stubProgramRepo) Create(_ context.Context, input repository.CreateWorkoutProgramInput) (*models.WorkoutProgram, error) {
	r.lastCreate = input
	return r.createResult, r.createErr
}

func (r *stubProgramRepo) ListByCoachID(_ context.Context, _ int64) ([]models.WorkoutProgram, error) {
	return r.listResult, r.listErr
}

func (r *stubProgramRepo) ListByUserID(_ context.Context, _ int64) ([]models.WorkoutProgram, error) {
	return r.listResult, r.listErr
}

func (r *stubProgramRepo) GetByID(_ context.Context, _ int64) (*models.WorkoutProgram, error) {
	return r.getResult, r.getErr
}

type stubProgramStorage struct {
	uploadURL      string
	uploadErr      error
	signedURL      string
	signedErr      error
	deleteErr      error
	lastUploadFile multipart.File
	lastFilename   string
	lastFolder     string
	lastDeletedURL string
	lastSignedURL  string
}

func (s *stubProgramStorage) UploadFile(_ context.Context, file multipart.File, filename string, folder string) (string, error) {
	s.lastUploadFile = file
	s.lastFilename = filename
	s.lastFolder = folder
	return s.uploadURL, s.uploadErr
}

func (s *stubProgramStorage) DeleteFile(_ context.Context, fileURL string) error {
	s.lastDeletedURL = fileURL
	return s.deleteErr
}

func (s *stubProgramStorage) GetSignedURL(_ context.Context, fileURL string) (string, error) {
	s.lastSignedURL = fileURL
	return s.signedURL, s.signedErr
}

type stubProgramUserRepo struct {
	user *models.User
	err  error
}

func (r *stubProgramUserRepo) GetByID(_ context.Context, _ int64) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

type stubRow struct {
	values []any
	err    error
}

func (r stubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch target := dest[i].(type) {
		case *int64:
			*target = r.values[i].(int64)
		case *int:
			*target = r.values[i].(int)
		case *string:
			*target = r.values[i].(string)
		case **string:
			*target = r.values[i].(*string)
		case *time.Time:
			*target = r.values[i].(time.Time)
		default:
			return errors.New("unsupported scan target")
		}
	}
	return nil
}

type stubDBTX struct {
	queryRowFn func(ctx context.Context, query string, args ...any) stubRow
}

func (db *stubDBTX) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (db *stubDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (db *stubDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return db.queryRowFn(ctx, query, args...)
}

type testMultipartFile struct {
	*bytes.Reader
}

func (f *testMultipartFile) Close() error {
	return nil
}

func newTestMultipartFile(content string) multipart.File {
	return &testMultipartFile{Reader: bytes.NewReader([]byte(content))}
}

var testTime = time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

func TestProgramServiceCreateProgramUploadsAndStoresProgram(t *testing.T) {
	programRepo := &stubProgramRepo{
		createResult: &models.WorkoutProgram{ID: 1, CoachID: 7, UserID: 42, SessionID: 99, FileURL: "https://storage/program.pdf"},
	}
	sessionRepo := repository.NewSessionRepository(&stubDBTX{
		queryRowFn: func(_ context.Context, query string, args ...any) stubRow {
			if strings.Contains(query, "FROM bookings") {
				return stubRow{values: []any{int64(99), int64(42), int64(7), testTime, 60, "completed", (*string)(nil), testTime, testTime}}
			}
			return stubRow{err: pgx.ErrNoRows}
		},
	})
	userRepo := &stubProgramUserRepo{user: &models.User{ID: 42, Role: "user"}}
	storage := &stubProgramStorage{uploadURL: "https://storage/program.pdf"}

	service := &ProgramService{
		programRepo:    programRepo,
		sessionRepo:    sessionRepo,
		userRepo:       userRepo,
		storageService: storage,
	}

	file := newTestMultipartFile("program-bytes")
	description := "Strength mesocycle"
	program, err := service.CreateProgram(context.Background(), 7, CreateProgramInput{
		UserID:      42,
		SessionID:   99,
		Title:       " Week 1 ",
		Description: &description,
		File:        file,
		Filename:    "program.pdf",
	})
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}

	if program.ID != 1 {
		t.Fatalf("expected program id 1, got %d", program.ID)
	}
	if storage.lastFolder != "programs" {
		t.Fatalf("expected programs folder, got %q", storage.lastFolder)
	}
	if !strings.HasSuffix(storage.lastFilename, ".pdf") {
		t.Fatalf("expected filename with pdf extension, got %q", storage.lastFilename)
	}
	if programRepo.lastCreate.Title != "Week 1" {
		t.Fatalf("expected trimmed title, got %q", programRepo.lastCreate.Title)
	}
	if programRepo.lastCreate.Description == nil || *programRepo.lastCreate.Description != "Strength mesocycle" {
		t.Fatalf("unexpected description: %+v", programRepo.lastCreate.Description)
	}
}

func TestProgramServiceCreateProgramDeletesUploadWhenInsertFails(t *testing.T) {
	programRepo := &stubProgramRepo{createErr: errors.New("insert failed")}
	sessionRepo := repository.NewSessionRepository(&stubDBTX{
		queryRowFn: func(_ context.Context, query string, args ...any) stubRow {
			if strings.Contains(query, "FROM bookings") {
				return stubRow{values: []any{int64(99), int64(42), int64(7), testTime, 60, "completed", (*string)(nil), testTime, testTime}}
			}
			return stubRow{err: pgx.ErrNoRows}
		},
	})
	userRepo := &stubProgramUserRepo{user: &models.User{ID: 42, Role: "user"}}
	storage := &stubProgramStorage{uploadURL: "https://storage/program.pdf"}

	service := &ProgramService{
		programRepo:    programRepo,
		sessionRepo:    sessionRepo,
		userRepo:       userRepo,
		storageService: storage,
	}

	file := newTestMultipartFile("program-bytes")
	_, err := service.CreateProgram(context.Background(), 7, CreateProgramInput{
		UserID:    42,
		SessionID: 99,
		Title:     "Week 1",
		File:      file,
		Filename:  "program.pdf",
	})
	if err == nil {
		t.Fatalf("expected create error")
	}
	if storage.lastDeletedURL != "https://storage/program.pdf" {
		t.Fatalf("expected uploaded file to be cleaned up, got %q", storage.lastDeletedURL)
	}
}

func TestProgramServiceCreateProgramSurfacesCleanupFailure(t *testing.T) {
	createErr := errors.New("insert failed")
	deleteErr := errors.New("delete failed")
	programRepo := &stubProgramRepo{createErr: createErr}
	sessionRepo := repository.NewSessionRepository(&stubDBTX{
		queryRowFn: func(_ context.Context, query string, args ...any) stubRow {
			if strings.Contains(query, "FROM bookings") {
				return stubRow{values: []any{int64(99), int64(42), int64(7), testTime, 60, "completed", (*string)(nil), testTime, testTime}}
			}
			return stubRow{err: pgx.ErrNoRows}
		},
	})
	userRepo := &stubProgramUserRepo{user: &models.User{ID: 42, Role: "user"}}
	storage := &stubProgramStorage{
		uploadURL: "https://storage/program.pdf",
		deleteErr: deleteErr,
	}

	service := &ProgramService{
		programRepo:    programRepo,
		sessionRepo:    sessionRepo,
		userRepo:       userRepo,
		storageService: storage,
	}

	file := newTestMultipartFile("program-bytes")
	_, err := service.CreateProgram(context.Background(), 7, CreateProgramInput{
		UserID:    42,
		SessionID: 99,
		Title:     "Week 1",
		File:      file,
		Filename:  "program.pdf",
	})
	if err == nil {
		t.Fatalf("expected create error")
	}
	if !errors.Is(err, createErr) {
		t.Fatalf("expected wrapped create error, got %v", err)
	}
	if !strings.Contains(err.Error(), "cleanup failed") || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("expected cleanup failure surfaced, got %v", err)
	}
	if storage.lastDeletedURL != "https://storage/program.pdf" {
		t.Fatalf("expected uploaded file cleanup to be attempted, got %q", storage.lastDeletedURL)
	}
}

func TestProgramServiceGetDownloadURLChecksAccess(t *testing.T) {
	programRepo := &stubProgramRepo{
		getResult: &models.WorkoutProgram{ID: 4, CoachID: 7, UserID: 42, FileURL: "https://storage/program.pdf"},
	}
	storage := &stubProgramStorage{signedURL: "https://signed/program.pdf"}

	service := &ProgramService{
		programRepo:    programRepo,
		storageService: storage,
	}

	url, err := service.GetDownloadURL(context.Background(), 42, "user", 4)
	if err != nil {
		t.Fatalf("GetDownloadURL: %v", err)
	}
	if url != "https://signed/program.pdf" {
		t.Fatalf("unexpected signed url: %q", url)
	}
	if storage.lastSignedURL != "https://storage/program.pdf" {
		t.Fatalf("unexpected source url: %q", storage.lastSignedURL)
	}

	_, err = service.GetDownloadURL(context.Background(), 99, "user", 4)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
