package postgres

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/Laps14/assets-tracker/stocks"
	"net/url"
	"os"
	"strings"
	"strconv"
	"time"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository() (*Repository, error) {
	dsn := fmt.Sprintf(`postgresql://%s:%s@127.0.0.1:1377/assets_tracker`, os.Getenv("POSTGRES_USER"), url.QueryEscape(os.Getenv("POSTGRES_PASSWORD")))

	db, err := sql.Open("pgx", dsn)

	if err != nil {
		return nil, fmt.Errorf("Error while creating connector: %v\n",err)
	}

	// Even after Docker created the container, there's a delay
	// before the container is up and running. Then, instead of
	// trying to ping only one time, I added a little timer befo
	// re it panics and quit.
	t := time.NewTimer(time.Second * 5)

	for {
		select {
		case <-t.C:
		return nil, fmt.Errorf("Couldn't ping into tracked_assets. Is PostgreSQL server running? %v\n", err)
		default:
			err = db.Ping()

			if err != nil {
				break
			}

			return &Repository{
				DB: db,
			}, nil
		}
	}

}

func (r *Repository) Select(ctx context.Context, id int64) (*stocks.Stock, error) {
	var stock stocks.Stock

	// QueryRow doesn't return error, so it's safe to only assign it's return value
	// to a variable and check for the err only after row.Scan returns
	row := r.DB.QueryRow("SELECT * FROM tracked_assets WHERE id=?", id)

	err := row.Scan(&stock.ID, &stock.Title, &stock.Description, &stock.StockVal, &stock.TargetVals, &stock.CompanyLogo)

	if err != nil {
		return nil, fmt.Errorf("Error while searching for id %v: %v\n", id, err)
	}

	return &stock, nil
}

func (r *Repository) SelectAll(ctx context.Context) ([]*stocks.Stock, error) {

	stockSlice := []*stocks.Stock{}

	rows, err := r.DB.Query("SELECT * FROM tracked_assets")

	if err != nil {
		return nil, fmt.Errorf("Error while querying for table tracked_assets\n")
	}
	defer rows.Close()

	for rows.Next() {
		stock := stocks.Stock{}
		var strTargetVals string

		if err := rows.Scan(&stock.ID, &stock.Title, &stock.Description, &stock.StockVal, &strTargetVals, &stock.CompanyLogo); err != nil {
			return nil, fmt.Errorf("Scanning book: %v", err)
		}

		strTargetVals = strings.ReplaceAll(strTargetVals, "{", "")
		strTargetVals = strings.ReplaceAll(strTargetVals, "}", "")

		for _, v := range strings.Split(strTargetVals, ",") {
			f, err := strconv.ParseFloat(v, 64)

			if err != nil {
				continue
			}

			stock.TargetVals = append(stock.TargetVals, f)
		}

		stockSlice = append(stockSlice, &stock)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error while querying for table tracked_assets\n")
	}

	return stockSlice, nil
}

// This function is a "provisory fix" (which means idk when I will change it).
// This looks weird (specially about the QueryRow using an INSERT)
// because pgx doesn't implement LastInsertId() function from Result interface.
// So... I thought about using the QueryRow and scannig the returned id from
// "INSERT INTO ... RETURNING id" statement and it worked (even if it's weird
// and wrong)
func (r *Repository) Insert(ctx context.Context, stock *stocks.Stock) (int64, error) {

	result := r.DB.QueryRow("INSERT INTO tracked_assets (title, description, stock_val, target_vals, company_logo) VALUES ($1, $2, $3, $4, $5) RETURNING id", stock.Title, stock.Description, stock.StockVal, stock.TargetVals, stock.CompanyLogo)

	var id int64

	if err := result.Scan(&id); err != nil {
		return 0, fmt.Errorf("Error while inserting %v into tracked_assets\n\n%v\n\n", stock.Title, err)
	}

	return id, nil
}

func (r *Repository) Update(ctx context.Context, stock *stocks.Stock) error {
	_, err := r.DB.Exec("UPDATE tracked_assets SET title = $1, description = $2, stock_val = $3, target_vals = $4, company_logo = $5 WHERE id = $6", stock.Title, stock.Description, stock.StockVal, stock.TargetVals, stock.CompanyLogo, stock.ID)

	if err != nil {
		return fmt.Errorf("Error while updating stock %v\n", stock.Title)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.Exec("DELETE FROM tracked_assets WHERE id = ?", id)

	if err != nil {
		return fmt.Errorf("Error while deleting id %v from tracked_assets\n", id)
	}

	return nil
}

func (r *Repository) DeleteAll(ctx context.Context) error {
	_, err := r.DB.Exec("DELETE FROM tracked_assets")

	if err != nil {
		return fmt.Errorf("Error while deleting stocks from tracked_assets\n")
	}

	return nil
}

func (r *Repository) Close() error {
	err := r.DB.Close()

	if err != nil {
		return fmt.Errorf("Couldn't close DB connection\n", err)
	}

	return nil
}
