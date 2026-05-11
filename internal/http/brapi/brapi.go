package brapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Laps14/assets-tracker/stocks"
	"net/http"
	// "strings"
	// "time"
	// "os"
	"io"
)

type token string
var header http.Header
var baseURL string
var tkn = token("token")

type stockRequest struct {
	Stocks []map[string]interface{} `json:"stocks"`
	Results []map[string]interface{} `json:"results"`
}

func NewContext(ctx context.Context, t string) context.Context {
	return context.WithValue(ctx, tkn, t)
}

func NewBrapiClient(ctx context.Context) {
	baseURL = "https://brapi.dev/api/quote/"

	header = map[string][]string{
		"Authorization": {"Bearer " + ctx.Value(tkn).(string)},
	}
}

func GetAllStocks(ctx context.Context) <-chan []stocks.Stock {

	stockchan := make(chan []stocks.Stock)

	go func () {
		stockSlice := make([]stocks.Stock, 0)

		req, err := http.NewRequest(http.MethodGet, baseURL + "list?type=stock", http.NoBody)

		if err != nil {
			fmt.Printf("\n%v\n", err)
			stockchan <- stockSlice
			return
		}

		req.Header = header

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			fmt.Printf("\n%v\n", err)
			stockchan <- stockSlice
			return
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)

		var jsonBody stockRequest

		if err := dec.Decode(&jsonBody); err != nil {
			fmt.Printf("\n%v\n", err)
			stockchan <- stockSlice
			return
		}

		for _, v := range jsonBody.Stocks {
			stockSlice = append(stockSlice, stocks.Stock{Title: v["stock"].(string), Description: v["name"].(string), StockVal: v["close"].(float64),
			CompanyLogo: v["logo"].(string)})
		}

		stockchan <- stockSlice
	}()

	return stockchan
}

func GetStock(ctx context.Context, stock stocks.Stock) (stocks.Stock, error) {
	var jsonBody stockRequest

	req, err := http.NewRequest(http.MethodGet, baseURL + stock.Title, http.NoBody)

	if err != nil {
		return stocks.Stock{}, fmt.Errorf("ERROR: %v\n", err)
	}

	req.Header = header

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return stocks.Stock{}, fmt.Errorf("ERROR: %v\n", err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	err = dec.Decode(&jsonBody)

	if err != nil {
		return stocks.Stock{}, fmt.Errorf("ERROR: %v\n", err)
	}

	updatedStock := stocks.Stock{
		Title: jsonBody.Results[0]["symbol"].(string),
		Description: jsonBody.Results[0]["longName"].(string),
		StockVal: jsonBody.Results[0]["regularMarketPrice"].(float64),
		CompanyLogo: jsonBody.Results[0]["logourl"].(string),
	}

	return updatedStock, nil
}

func GetStockCompanyLogo(ctx context.Context, stock stocks.Stock) ([]byte, error) {

	req, err := http.NewRequest(http.MethodGet, stock.CompanyLogo, http.NoBody)
	
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	header.Add("Accept", "image/svg+xml")
	req.Header = header

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	img_bytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	defer resp.Body.Close()

	return img_bytes, nil
}
