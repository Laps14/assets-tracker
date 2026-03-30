package stocks

import "context"

type Reader interface {
	Select(ctx context.Context, id uint64) (*Stock, error)
	SelectAll(ctx context.Context) ([]*Stock, error)
}

type Writer interface {
	Insert(ctx context.Context, stock *Stock) (uint64, error)
	Update(ctx context.Context, stock *Stock) error
	Delete(ctx context.Context, id uint64) error
}

type Repository interface {
	Reader
	Writer
	Close() error
}
