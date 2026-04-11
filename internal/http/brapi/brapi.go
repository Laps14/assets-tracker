package brapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Laps14/assets-tracker/stocks"
	"golang.org/x/sys/unix"
	"net/http"
	"os/signal"
	"strings"
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

func GetAllStocks(ctx context.Context) (context.Context, <-chan []stocks.Stock) {
	signal.Reset()

	ctx, stop := signal.NotifyContext(
		ctx,
		unix.SIGINT, unix.SIGQUIT, unix.SIGHUP,
	)

	req, err := http.NewRequest(http.MethodGet, baseURL + "list?type=stock", http.NoBody)

	if err != nil {
		fmt.Errorf("ERROR: %v", err)
	}

	req.Header = header

	stockchan := make(chan []stocks.Stock)

	go func(sc chan []stocks.Stock) {
		defer stop()
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}

		dec := json.NewDecoder(resp.Body)
		stockSlice := make([]stocks.Stock, 0)

		var jsonBody stockRequest

		if err := dec.Decode(&jsonBody); err != nil {
			fmt.Println("ERROR: ", err)
			return
		}

		for _, v := range jsonBody.Stocks {
			stockSlice = append(stockSlice, stocks.Stock{Title: v["stock"].(string), Description: v["name"].(string), StockVal: v["close"].(float64)})
		}

		sc <- stockSlice
	}(stockchan)

	return ctx, stockchan
}

func GetStocks(ctx context.Context, stockSlice []stocks.Stock) ([]stocks.Stock, error) {
	var (
		stock_tickers strings.Builder
		ith_stock = 0
		updateStocks = make([]stocks.Stock, 0)
		jsonBody stockRequest
	)

	for ; ith_stock < len(stockSlice); ith_stock++ {
		stock_tickers.WriteString(stockSlice[ith_stock].Title)

		req, err := http.NewRequest(http.MethodGet, baseURL + stock_tickers.String(), http.NoBody)

		if err != nil {
			return nil, fmt.Errorf("ERROR: %v\n", err)
		}

		req.Header = header

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("ERROR: %v\n", err)
		}

		dec := json.NewDecoder(resp.Body)

		err = dec.Decode(&jsonBody)

		if err != nil {
			return nil, fmt.Errorf("ERROR: %v\n", err)
		}

		for _, stock := range jsonBody.Results {
			// fmt.Printf("%d\t%v\n", i, stock)
			tempStock := stocks.Stock{
				Title: stock["symbol"].(string),
				Description: stock["longName"].(string),
				StockVal: stock["regularMarketPrice"].(float64),
			}
			updateStocks = append(updateStocks, tempStock)
		}

		stock_tickers.Reset()
		resp.Body.Close()
	}

	return updateStocks, nil
}
//
// func GetStocks(ctx context.Context, stockSlice []stocks.Stock) ([]stocks.Stock, error) {
// 	var (
// 		stock_tickers strings.Builder
// 		ith_stock = 0
// 		updateStocks = make([]stocks.Stock, 0)
// 		jsonBody stockRequest
// 	)
//
// 	for ; ith_stock < len(stockSlice) - 1; ith_stock++ {
// 		stock_tickers.WriteString(stockSlice[ith_stock].Title + ",")
// 	}
// 	defer stock_tickers.Reset()
//
// 	stock_tickers.WriteString(stockSlice[ith_stock].Title)
//
// 	req, err := http.NewRequest(http.MethodGet, baseURL + stock_tickers.String(), http.NoBody)
//
// 	if err != nil {
// 		return nil, fmt.Errorf("ERROR: %v\n", err)
// 	}
//
// 	req.Header = header
//
// 	resp, err := http.DefaultClient.Do(req)
//
// 	if err != nil {
// 		return nil, fmt.Errorf("ERROR: %v\n", err)
// 	}
// 	defer resp.Body.Close()
//
// 	dec := json.NewDecoder(resp.Body)
//
// 	err = dec.Decode(&jsonBody)
//
// 	if err != nil {
// 		return nil, fmt.Errorf("ERROR: %v\n", err)
// 	}
//
// 	for _, stock := range jsonBody.Results {
// 		// fmt.Printf("%d\t%v\n", i, stock)
// 		tempStock := stocks.Stock{
// 			Title: stock["symbol"].(string),
// 			Description: stock["longName"].(string),
// 			StockVal: stock["regularMarketPrice"].(float64),
// 		}
// 		updateStocks = append(updateStocks, tempStock)
// 		fmt.Println(updateStocks)
// 	}
//
// 	return updateStocks, nil
// }
