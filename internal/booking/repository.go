package booking

import (
	"context"
)

type Repository interface {
	// ListShows returns every show, in creation order.
	ListShows(ctx context.Context) ([]Show, error)
	// InsertShow saves a new show.
	InsertShow(ctx context.Context, s Show) error
}
