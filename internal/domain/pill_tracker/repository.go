package pill_tracker

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, tracker *PillTracker) error
	Update(ctx context.Context, tracker *PillTracker) error
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*PillTracker, error)
	FindByID(ctx context.Context, id uuid.UUID) (*PillTracker, error)
	FindAllActive(ctx context.Context) ([]*PillTracker, error)
}
