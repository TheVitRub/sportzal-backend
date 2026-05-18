package interfaces

import (
	"context"

	"workout-app/backend/internal/domain/model"
)

type StateRepository interface {
	View(ctx context.Context, fn func(state model.State) error) error
	Update(ctx context.Context, fn func(state *model.State) error) error
}
