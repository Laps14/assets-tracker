package stocks

import (
	"context"
	"net/http"
)

type UseCase interface {
	Create(context.Context, string, string, Sector, float32, float32) (*Stock, err)
	Update(context.Context, string, string, Sector, float32, float32) error
	Delete(context.Context, uint64) error
	Get(context.Context, uint64) (*Stock, error)
	List(context.Context) ([]*Stock, error)
}

type Service struct {
	Repo Repository
}

func NewService(r Repository) *Service {
	return &Service{
		Repo: r
	}
}

func (s *Service) Create(ctx context.Context, title, description string, sector Sector, total_supply, stock_val float32) (*Stock, err) {
	stock := Stock{
		Title: title
		Description: description
		Sect: sector
		TotalSupply: total_supply
		StockVal: stock_val
	}

	id, err := s.Repo.Insert(ctx, &stock)

	if err != nil {
		fmt.Errorf("ERROR: %v", err)
		return nil, err
	}

	stock.ID = id

	return &stock, nil
}

func (s *Service) Update(ctx context.Context, title, description string, sector Sector, total_supply, stock_val float32) error {
	stock := Stock{
		Title: title
		Description: description
		Sect: sector
		TotalSupply: total_supply
		StockVal: stock_val
	}

	err := s.Repo.Update(ctx, &stock)

	if err != nil {
		fmt.Errorf("ERROR: %v", err)
		return err
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id uint64) error {
	err := s.Repo.Delete(ctx, id)

	if err != nil {
		fmt.Errorf("ERROR: %v", err)
		return err
	}

	return nil
}

func (s *Service) Get(ctx context.Context, id uint64) (*Stock, error) {
	stock, err := s.Repo.Select(ctx, id)

	if err != nil {
		fmt.Errorf("ERROR: %v", err)
		return nil, err
	}

	return &stock, nil
}

func (s *Service) List(ctx context.Context) ([]*Stock, error) {
	stocks, err := s.Repo.SelectAll(ctx)

	if err != nil {
		fmt.Errorf("ERROR: %v", err)
		return nil, err
	}

	return &stocks, nil
}
