package brapi

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"net/http"
	"os/signal"
	"github.com/Laps14/assets-tracker/stocks"
)

type token string
var header http.Header
var baseURL string
var tkn = token("token")

type stockRequest struct {
	Stocks []map[string]interface{} `json:"stocks"`
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

func GetStocks(ctx context.Context) (context.Context, <-chan []stocks.Stock) {
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
