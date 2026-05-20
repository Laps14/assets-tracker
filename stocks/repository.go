package stocks

import "context"

type Reader interface {
	Select(ctx context.Context, id int64) (*Stock, error)
	SelectAll(ctx context.Context) ([]*Stock, error)
}

type Writer interface {
	Insert(ctx context.Context, stock *Stock) (int64, error)
	Update(ctx context.Context, stock *Stock) error
	Delete(ctx context.Context, id int64) error
	DeleteAll(ctx context.Context) error
}

type Repository interface {
	Reader
	Writer
	Close() error
}
