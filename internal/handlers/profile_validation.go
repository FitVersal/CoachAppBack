package handlers

import (
	"strings"
)

var allowedGenders = map[string]struct{}{
	"male":              {},
	"female":            {},
	"other":             {},
	"prefer_not_to_say": {},
}

var allowedFitnessLevels = map[string]struct{}{
	"beginner":     {},
	"intermediate": {},
	"advanced":     {},
}

func validateUserOnboardingRequest(req userOnboardingRequest) string {
	if strings.TrimSpace(req.FullName) == "" {
		return "full_name is required"
	}
	if req.Age <= 0 {
		return "age must be greater than 0"
	}
	if err := validateGender(req.Gender); err != "" {
		return err
	}
	if req.HeightCM <= 0 {
		return "height_cm must be greater than 0"
	}
	if req.WeightKG <= 0 {
		return "weight_kg must be greater than 0"
	}
	if err := validateFitnessLevel(req.FitnessLevel); err != "" {
		return err
	}
	if len(req.Goals) == 0 {
		return "goals must contain at least one item"
	}
	for _, goal := range req.Goals {
		if strings.TrimSpace(goal) == "" {
			return "goals must not contain empty values"
		}
	}
	return ""
}

func validateCoachOnboardingRequest(req coachOnboardingRequest) string {
	if strings.TrimSpace(req.FullName) == "" {
		return "full_name is required"
	}
	if strings.TrimSpace(req.Bio) == "" {
		return "bio is required"
	}
	if len(req.Specializations) == 0 {
		return "specializations must contain at least one item"
	}
	for _, specialization := range req.Specializations {
		if strings.TrimSpace(specialization) == "" {
			return "specializations must not contain empty values"
		}
	}
	if len(req.Certifications) == 0 {
		return "certifications must contain at least one item"
	}
	for _, certification := range req.Certifications {
		if strings.TrimSpace(certification) == "" {
			return "certifications must not contain empty values"
		}
	}
	if req.ExperienceYears < 0 {
		return "experience_years must be 0 or greater"
	}
	if req.HourlyRate < 0 {
		return "hourly_rate must be 0 or greater"
	}
	return ""
}

func validateUserProfileUpdateRequest(req updateUserProfileRequest) string {
	if req.FullName != nil && strings.TrimSpace(*req.FullName) == "" {
		return "full_name must not be empty"
	}
	if req.Age != nil && *req.Age <= 0 {
		return "age must be greater than 0"
	}
	if req.Gender != nil {
		if err := validateGender(*req.Gender); err != "" {
			return err
		}
	}
	if req.HeightCM != nil && *req.HeightCM <= 0 {
		return "height_cm must be greater than 0"
	}
	if req.WeightKG != nil && *req.WeightKG <= 0 {
		return "weight_kg must be greater than 0"
	}
	if req.FitnessLevel != nil {
		if err := validateFitnessLevel(*req.FitnessLevel); err != "" {
			return err
		}
	}
	if req.Goals != nil {
		for _, goal := range *req.Goals {
			if strings.TrimSpace(goal) == "" {
				return "goals must not contain empty values"
			}
		}
	}
	if req.MedicalConditions != nil && strings.TrimSpace(*req.MedicalConditions) == "" {
		return "medical_conditions must not be empty"
	}
	return ""
}

func validateCoachProfileUpdateRequest(req updateCoachProfileRequest) string {
	if req.FullName != nil && strings.TrimSpace(*req.FullName) == "" {
		return "full_name must not be empty"
	}
	if req.Bio != nil && strings.TrimSpace(*req.Bio) == "" {
		return "bio must not be empty"
	}
	if req.Specializations != nil {
		for _, specialization := range *req.Specializations {
			if strings.TrimSpace(specialization) == "" {
				return "specializations must not contain empty values"
			}
		}
	}
	if req.Certifications != nil {
		for _, certification := range *req.Certifications {
			if strings.TrimSpace(certification) == "" {
				return "certifications must not contain empty values"
			}
		}
	}
	if req.ExperienceYears != nil && *req.ExperienceYears < 0 {
		return "experience_years must be 0 or greater"
	}
	if req.HourlyRate != nil && *req.HourlyRate < 0 {
		return "hourly_rate must be 0 or greater"
	}
	return ""
}

func validateGender(gender string) string {
	if _, ok := allowedGenders[strings.TrimSpace(gender)]; !ok {
		return "gender must be one of: male, female, other, prefer_not_to_say"
	}
	return ""
}

func validateFitnessLevel(level string) string {
	if _, ok := allowedFitnessLevels[strings.TrimSpace(level)]; !ok {
		return "fitness_level must be one of: beginner, intermediate, advanced"
	}
	return ""
}
