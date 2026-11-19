package contracts

import (
	"context"

	"goravel/packages/feedback/models"
)

// Service is the portable feedback API other Goravel apps can depend on.
type Service interface {
	List(ctx context.Context, page, pageSize int, status *uint8) ([]models.Feedback, int64, error)
	Get(ctx context.Context, id uint) (*models.Feedback, error)
	Create(ctx context.Context, contact, content string) (*models.Feedback, error)
	MarkDone(ctx context.Context, id uint) (*models.Feedback, error)
	Delete(ctx context.Context, id uint) error
}
