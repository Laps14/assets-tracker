package brapi

import (
	"context"
	"fmt"
	// "github.com/Laps14/assets-tracker/config"
	"net/http"
	"encoding/json"
	"os"
	"os/signal"
	"time"
	"golang.org/x/sys/unix"
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

func GetStocks(ctx context.Context) <-chan bool{
	c := make(chan bool)

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

	go func() {
		defer stop()
		for _, ok := <-ctx.Done(); !ok; _, ok = <-ctx.Done() {
			resp, err := http.DefaultClient.Do(req)

			time.Sleep(5 * time.Second)

			if err != nil {
				fmt.Errorf("ERROR: %v", err)
				c <- false
				return
			}

			body := json.NewDecoder(resp.Body)

			logfile, err := os.Create("/home/lucas/.assets-tracker/logfile.json")

			if err != nil {
				fmt.Errorf("ERROR: %v", err)
				c <- false
				return
			}

			for {

				var v map[string]interface{}

				if err := body.Decode(&v); err != nil {
					fmt.Errorf("ERROR: %v", err)
					break
				}

				for k, vv := range v {
					fmt.Fprintf(logfile, "%v -> %v\n", k, vv)
				}
			}
			defer resp.Body.Close()

			c <- true
			return
		}
		// select {
		// case <-ctx.Done():
		// 	for i := 0; i < 10; i++ {
		// 		fmt.Println("SIGNAL")
		// 	}
		// 	c <- false
		// default:
		//
		// 	resp, err := http.DefaultClient.Do(req)
		//
		// 	time.Sleep(5 * time.Second)
		//
		// 	if err != nil {
		// 		fmt.Errorf("ERROR: %v", err)
		// 		c <- false
		// 	}
		//
		// 	body := json.NewDecoder(resp.Body)
		//
		// 	logfile, err := os.Create("/home/lucas/.assets-tracker/logfile.json")
		//
		// 	if err != nil {
		// 		fmt.Errorf("ERROR: %v", err)
		// 		c <- false
		// 	}
		//
		// 	for {
		//
		// 		var v map[string]interface{}
		//
		// 		if err := body.Decode(&v); err != nil {
		// 			fmt.Errorf("ERROR: %v", err)
		// 			break
		// 		}
		//
		// 		for k, vv := range v {
		// 			fmt.Fprintf(logfile, "%v -> %v\n", k, vv)
		// 		}
		// 	}
		// 	defer resp.Body.Close()
		//
		// 	c <- true
		// }
	}()

	return c
}
