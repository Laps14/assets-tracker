package brapi

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"net/http"
	"os/signal"
)

type token string
var header http.Header
var baseURL string
var tkn = token("token")

func NewContext(ctx context.Context, t string) context.Context {
	return context.WithValue(ctx, tkn, t)
}

func NewBrapiClient(ctx context.Context) {
	baseURL = "https://brapi.dev/api/quote/"

	header = map[string][]string{
		"Authorization": {"Bearer " + ctx.Value(tkn).(string)},
	}
}

func GetStocks(ctx context.Context) (context.Context, <-chan bool) {
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

	stockchan := make(chan bool)

	go func(sc chan bool) {
		defer stop()
		resp, err := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if err != nil {
			fmt.Errorf("ERROR: %v", err)
			return
		}

		dec := json.NewDecoder(resp.Body)

		for {
			var v map[string]interface{}

			if err := dec.Decode(&v); err != nil {
				return
			}

			fmt.Printf("%#v\n", v)
		}

		sc <- true
	}(stockchan)

	return ctx, stockchan
}
