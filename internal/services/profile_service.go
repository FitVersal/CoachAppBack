package services

import (
	"context"

	"github.com/saeid-a/CoachAppBack/internal/models"
	"github.com/saeid-a/CoachAppBack/internal/repository"
)

type UserProfileUpdater interface {
	UpdatePartial(ctx context.Context, userID int64, req repository.UpdateUserProfileInput) (*models.UserProfile, error)
}

type CoachProfileUpdater interface {
	UpdatePartial(ctx context.Context, userID int64, req repository.UpdateCoachProfileInput) (*models.CoachProfile, error)
}

type ProfileService struct {
	userProfileRepo  UserProfileUpdater
	coachProfileRepo CoachProfileUpdater
}

func NewProfileService(userProfileRepo UserProfileUpdater, coachProfileRepo CoachProfileUpdater) *ProfileService {
	return &ProfileService{
		userProfileRepo:  userProfileRepo,
		coachProfileRepo: coachProfileRepo,
	}
}

func (s *ProfileService) UpdateUserProfile(ctx context.Context, userID int64, req repository.UpdateUserProfileInput) (*models.UserProfile, error) {
	return s.userProfileRepo.UpdatePartial(ctx, userID, req)
}

func (s *ProfileService) UpdateCoachProfile(ctx context.Context, userID int64, req repository.UpdateCoachProfileInput) (*models.CoachProfile, error) {
	return s.coachProfileRepo.UpdatePartial(ctx, userID, req)
}
