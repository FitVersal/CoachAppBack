package services

import (
	"context"
	"sort"
	"strings"

	"github.com/saeid-a/CoachAppBack/internal/models"
)

type CoachMatcher interface {
	ListAll(ctx context.Context) ([]models.CoachProfile, error)
}

type MatchmakingService struct {
	coachRepo CoachMatcher
}

func NewMatchmakingService(coachRepo CoachMatcher) *MatchmakingService {
	return &MatchmakingService{coachRepo: coachRepo}
}

func (s *MatchmakingService) GetMatchedCoaches(
	ctx context.Context,
	userProfile *models.UserProfile,
	limit int,
) ([]models.CoachWithScore, error) {
	coaches, err := s.coachRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	matched := make([]models.CoachWithScore, 0, len(coaches))
	for _, coach := range coaches {
		matched = append(matched, models.CoachWithScore{
			CoachProfile: coach,
			MatchScore:   calculateMatchScore(userProfile, &coach),
		})
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].MatchScore == matched[j].MatchScore {
			return floatValue(matched[i].Rating) > floatValue(matched[j].Rating)
		}
		return matched[i].MatchScore > matched[j].MatchScore
	})

	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	return matched, nil
}

func calculateMatchScore(userProfile *models.UserProfile, coach *models.CoachProfile) int {
	score := 0
	goalTags := goalAliases(userProfile)
	coachSpecs := normalizeValues(coach.Specializations)

	for _, aliases := range goalTags {
		for _, alias := range aliases {
			if _, ok := coachSpecs[alias]; ok {
				score += 40
				break
			}
		}
	}

	if floatValue(coach.Rating) > 4.0 {
		score += 20
	}
	if intValue(coach.ExperienceYears) > 3 {
		score += 15
	}
	if len(sliceValue(coach.Certifications)) > 0 {
		score += 10
	}
	if budget := floatValue(userBudget(userProfile)); budget > 0 && floatValue(coach.HourlyRate) <= budget {
		score += 15
	}

	return score
}

func goalAliases(userProfile *models.UserProfile) map[string][]string {
	goals := sliceValue(nil)
	if userProfile != nil {
		goals = sliceValue(userProfile.Goals)
	}

	mapped := make(map[string][]string, len(goals))
	for _, goal := range goals {
		switch normalize(goal) {
		case "weight_loss", "fat_loss":
			mapped["weight_loss"] = []string{"weight_loss", "fat_loss"}
		case "muscle_gain":
			mapped["muscle_gain"] = []string{"muscle_gain", "bodybuilding", "strength_training"}
		case "strength":
			mapped["strength"] = []string{"strength", "strength_training"}
		case "flexibility", "mobility":
			mapped["flexibility"] = []string{"flexibility", "mobility", "yoga"}
		default:
			if key := normalize(goal); key != "" {
				mapped[key] = []string{key}
			}
		}
	}

	return mapped
}

func normalizeValues(values *[]string) map[string]struct{} {
	normalized := make(map[string]struct{})
	for _, value := range sliceValue(values) {
		if key := normalize(value); key != "" {
			normalized[key] = struct{}{}
		}
	}
	return normalized
}

func normalize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func sliceValue(values *[]string) []string {
	if values == nil {
		return nil
	}
	return *values
}

func floatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func userBudget(userProfile *models.UserProfile) *float64 {
	if userProfile == nil {
		return nil
	}
	return userProfile.MaxHourlyRate
}
