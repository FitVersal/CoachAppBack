package services

import (
	"context"
	"testing"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type stubCoachMatcher struct {
	coaches []models.CoachProfile
}

func (s *stubCoachMatcher) ListAll(_ context.Context) ([]models.CoachProfile, error) {
	return s.coaches, nil
}

func TestGetMatchedCoachesSortsByScoreThenRating(t *testing.T) {
	goals := []string{"muscle_gain", "weight_loss"}
	budget := 50.0
	service := NewMatchmakingService(&stubCoachMatcher{
		coaches: []models.CoachProfile{
			buildCoachProfile(11, []string{"bodybuilding", "strength_training"}, 4.8, 6, 45, []string{"NASM"}),
			buildCoachProfile(12, []string{"weight_loss"}, 4.9, 4, 49, nil),
			buildCoachProfile(13, []string{"yoga"}, 5.0, 10, 40, []string{"RYT"}),
		},
	})

	matched, err := service.GetMatchedCoaches(context.Background(), &models.UserProfile{
		Goals:         &goals,
		MaxHourlyRate: &budget,
	}, 3)
	if err != nil {
		t.Fatalf("GetMatchedCoaches: %v", err)
	}

	if got := len(matched); got != 3 {
		t.Fatalf("expected 3 coaches, got %d", got)
	}
	if matched[0].UserID != 11 || matched[0].MatchScore != 100 {
		t.Fatalf("expected coach 11 with score 100 first, got coach %d with score %d", matched[0].UserID, matched[0].MatchScore)
	}
	if matched[1].UserID != 12 || matched[1].MatchScore != 90 {
		t.Fatalf("expected coach 12 with score 90 second, got coach %d with score %d", matched[1].UserID, matched[1].MatchScore)
	}
	if matched[2].UserID != 13 || matched[2].MatchScore != 60 {
		t.Fatalf("expected coach 13 with score 60 third, got coach %d with score %d", matched[2].UserID, matched[2].MatchScore)
	}
}

func TestGetMatchedCoachesAppliesLimit(t *testing.T) {
	goals := []string{"weight_loss"}
	service := NewMatchmakingService(&stubCoachMatcher{
		coaches: []models.CoachProfile{
			buildCoachProfile(1, []string{"weight_loss"}, 4.5, 5, 60, nil),
			buildCoachProfile(2, []string{"yoga"}, 4.9, 7, 50, nil),
		},
	})

	matched, err := service.GetMatchedCoaches(context.Background(), &models.UserProfile{Goals: &goals}, 1)
	if err != nil {
		t.Fatalf("GetMatchedCoaches: %v", err)
	}
	if got := len(matched); got != 1 {
		t.Fatalf("expected 1 coach, got %d", got)
	}
	if matched[0].UserID != 1 {
		t.Fatalf("expected top coach to be 1, got %d", matched[0].UserID)
	}
}

func TestGetMatchedCoachesBudgetBonusRequiresPreference(t *testing.T) {
	goals := []string{"weight_loss"}
	service := NewMatchmakingService(&stubCoachMatcher{
		coaches: []models.CoachProfile{
			buildCoachProfile(1, []string{"weight_loss"}, 4.2, 4, 40, nil),
			buildCoachProfile(2, []string{"weight_loss"}, 4.2, 4, 80, nil),
		},
	})

	budget := 50.0
	matched, err := service.GetMatchedCoaches(context.Background(), &models.UserProfile{
		Goals:         &goals,
		MaxHourlyRate: &budget,
	}, 2)
	if err != nil {
		t.Fatalf("GetMatchedCoaches: %v", err)
	}

	if matched[0].MatchScore != matched[1].MatchScore+15 {
		t.Fatalf("expected budget bonus gap of 15, got %d vs %d", matched[0].MatchScore, matched[1].MatchScore)
	}
}

func TestGoalAliasesHandleDocumentedSynonyms(t *testing.T) {
	goals := []string{"mobility", "weight_loss"}
	service := NewMatchmakingService(&stubCoachMatcher{
		coaches: []models.CoachProfile{
			buildCoachProfile(1, []string{"fat_loss", "flexibility"}, 0, 0, 999, nil),
		},
	})

	matched, err := service.GetMatchedCoaches(context.Background(), &models.UserProfile{
		Goals: &goals,
	}, 1)
	if err != nil {
		t.Fatalf("GetMatchedCoaches: %v", err)
	}

	if got := matched[0].MatchScore; got != 80 {
		t.Fatalf("expected synonym goal match score 80, got %d", got)
	}
}

func buildCoachProfile(userID int64, specs []string, rating float64, experience int, rate float64, certs []string) models.CoachProfile {
	return models.CoachProfile{
		UserID:             userID,
		Specializations:    &specs,
		Rating:             &rating,
		ExperienceYears:    &experience,
		HourlyRate:         &rate,
		Certifications:     &certs,
		OnboardingComplete: true,
	}
}
