package handlers

import "github.com/saeid-a/CoachAppBack/internal/models"

const (
	defaultPageLimit = 10
	maxPageLimit     = 50
)

func buildPaginationMeta(page, limit, total int) models.PaginationMeta {
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return models.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
