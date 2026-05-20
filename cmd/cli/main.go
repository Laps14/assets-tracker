package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Laps14/assets-tracker/config"
	"github.com/Laps14/assets-tracker/internal/http/brapi"
	"github.com/Laps14/assets-tracker/notifications"
	"github.com/Laps14/assets-tracker/stocks/postgres"
	"github.com/Laps14/assets-tracker/stocks"
	"github.com/Laps14/assets-tracker/tty"
	"golang.org/x/sys/unix"
	"maps"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	help_templ = "\n\x1b[1mh, help\x1b[0m\n\t Apresenta esta mensagem de ajuda.\n\n" +
		"\x1b[1ml, list\x1b[0m\n\t Lista todas as ações possíveis.\n\n" +
		"\x1b[1m--tracked\x1b[0m\n\t Opção usada com list para listar apenas as ações monitoradas.\n\n" +
		"\x1b[1mw, write STOCK [STOCK...]\x1b[0m\n\t Salva a(s) STOCK(s) para serem monitoradas. Para mais de uma ação, separá-las por espaço.\n\n" +
		"\x1b[1mSTOCK\x1b[0m\n\t O código/sigla de uma ação. Caso o tipo (3, 4, 5, 3F etc.) seja omitido, por padrão, a ação com indicador 4F (ou 3F, caso haja) será utilizada.\n\n"
)

var loading_chars = [...]string{"\u2819", "\u2838", "\u28B0", "\u28E0", "\u28C4", "\u2846", "\u2807"}

var chanUpdateStocksRoutine chan bool

func main() {

	term, err := tty.NewTtyConfig()

	if err != nil {
		panic(1)
	}

	term.InitialTtyPrompt()

	conf, err := config.InitializeConfig()

	if err != nil {
		panic(1)
	}

	ctx := context.Background()
	mainSignals := make(chan os.Signal) // Will be used to handle signals

	conf.CheckDirectories(ctx, term)

	ctx = brapi.NewContext(ctx, conf.AccessToken)

	brapi.NewBrapiClient(ctx)

	stock_hmap := stocks.NewStocks()

	repository, err := postgres.NewRepository()

	if err != nil {
		fmt.Printf("%v", err)
		panic(1)
	}
	defer repository.Close()

	service := stocks.NewService(repository)

	trackedStocksChan, tempMap := startTrackerNotifications(ctx, conf, service)

	var ith_char int = 0

	stockChan := brapi.GetAllStocks(ctx)

	term.DisableCursor()

	MainOuterLoop:
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("\x1b[2K\r")
				return
			case stockSlice := <-stockChan:
				if len(stockSlice) == 0 {
					fmt.Printf("\n\nERROR 0\n\n")
					break MainOuterLoop
				}
				stock_hmap.InsertAll(stockSlice)
				break MainOuterLoop
			default:
				loadingLine(&ith_char, term)
			}
		}

	term.EnableCursor()

	term.ShutdownTtyRoutine(mainSignals) // Shutdown goroutine

	chanUpdateStocksRoutine = updateStocksRoutine(ctx, stock_hmap)

	// Some commands call signal.Reset because it will be able
	// to cancel "itself" without quitting the tracker.
	// After the execution of a command, the main will be able
	// to receive signals again.

	for {
		getCmd(ctx, term, stock_hmap, trackedStocksChan, conf, tempMap)
		signal.Notify(mainSignals, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGQUIT)
	}
}

func getCmd(ctx context.Context, term *tty.Tty, s *stocks.Stocks, trackedStocksChan chan<- *stocks.Stock, conf *config.Config, tempMap *map[string]*stocks.Stock) {

	fmt.Print("\x1b[1m\u276F\u276F \x1b[0m")

	line, err := term.ReadLine()

	if err != nil {
		fmt.Errorf("Error while reading the input")
		panic(1)
	}

	if len(line) == 0 {
		fmt.Println("- Operação inválida")
		return
	}

	args := strings.Fields(line)

	switch args[0] {
	case "w", "write":
		if len(args) == 1 {
			fmt.Println("É necessário informar ao menos 1 uma STOCK.")
			return
		}

		var str string
		str = strings.Join(args[1:], " ")

		reg := regexp.MustCompile(`\b[a-zA-Z]{4}(\d{0,2}|\d{0,2}[Bb]?[Ff]?)[[:blank:]][\-\+]?\d+(?:\.\d{1,2})?(?:\%\B)?`)

		errs := slices.Compact(reg.Split(str, -1)) // Will take off repeated errors
		blankSpaces_errs := strings.TrimSpace(strings.Join(errs, " ")) // Will trim annoying spaces and empty strings

		if len(blankSpaces_errs) > 0 {
			for _, e := range errs {
				if trimmed := strings.TrimSpace(e); trimmed != "" {
					fmt.Printf("Argumento inválido próximo a %q.\n", e)
				}
			}
			return
		}

		for i, r := 0, args[1:]; i < len(r); i += 2 {
			stock, err := s.Search(r[i])

			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
				continue
			}

			stockWithTargetVal, err := treatTargetValue(stock, r[i+1])

			if err != nil {
				fmt.Println(err)
				return
			}

			trackedStocksChan <- stockWithTargetVal
		}
		return
	case "l", "list":

		if len(args) > 1 {
			if slices.Contains(args, "--tracked") {
				for _, v := range *tempMap {
					v.String()
				}
				return
			} else {
				var str string
				str = strings.Join(args[1:], " ")

				// Checks if the input line has any non-word char
				// It can't be a simple \W because then the spaces
				// would be interpreted as errors.
				reg := regexp.MustCompile(`\b[a-zA-Z]{4}(?:\d{0,2}|\d{1,2}[Ff])\b`) 

				errs := slices.Compact(reg.Split(str, -1)) // Will take off repeated errors
				blankSpaces_errs := strings.TrimSpace(strings.Join(errs, " ")) // Will trim annoying spaces and empty strings

				if len(blankSpaces_errs) > 0 {
					for _, e := range errs {
						if trimmed := strings.TrimSpace(e); trimmed != "" {
							fmt.Printf("\nArgumento inválido próximo a %q.\n", e)
						}
					}
					return
				}

				stockSlice := s.SearchAll(args[1:])

				for _, stock := range stockSlice {
					stock.String()
				}

				return
			}
		}

		chanUpdateStocksRoutine <- true

		var ith_char int = 0

		term.DisableCursor()
		defer term.EnableCursor()

		for {
			select {
			case result := <-chanUpdateStocksRoutine:
				if result == true {
					sorted := s.ToSlice()
					sortByStock(sorted)
					print2Cols(sorted, term)
					return
				}
			default:
				loadingLine(&ith_char, term)
			}
		}

	case "s", "show":
		var str string
		str = strings.Join(args[1:], " ")

		// Checks if the input line has any non-word char
		// It can't be a simple \W because then the spaces
		// would be interpreted as errors.
		reg := regexp.MustCompile(`\b[a-zA-Z]{4}(?:\d{0,2}|\d{1,2}[Ff])\b`) 

		searched_stock := reg.FindString(str)

		errs := slices.Compact(reg.Split(str, -1)) // Will take off repeated errors
		blankSpaces_errs := strings.TrimSpace(strings.Join(errs, " ")) // Will trim annoying spaces and empty strings

		if len(blankSpaces_errs) > 0 {
			for _, e := range errs {
				if trimmed := strings.TrimSpace(e); trimmed != "" {
					fmt.Printf("\nArgumento inválido próximo a %q.\n", e)
				}
			}
			return
		}
		err := showStock(ctx, searched_stock, s, conf, term)

		if err != nil {
			fmt.Printf("\n\n%v\n\n", err)
			return
		}
	case "h", "help":
		templ, err := template.New("help").Parse(help_templ)
		
		if err != nil {
			fmt.Errorf("Não foi possível apresentar a mensagem de ajuda")
		}

		templ.Execute(os.Stdout, nil)
	default:
		fmt.Println("- Operação inválida")
	}
}

func loadingLine(ith_char *int, term *tty.Tty) {

	var strBuilder strings.Builder

	if *ith_char >= len(loading_chars) { *ith_char = 0 }
	fmt.Fprintf(&strBuilder, "Carregando %s", loading_chars[*ith_char])
	fmt.Print(strBuilder.String())
	time.Sleep(50 * time.Millisecond)
	fmt.Print("\x1b[2K")
	term.MoveCurPos(term.Rows(), 1)
	*ith_char++
}

func sortByStock(stockSlice []*stocks.Stock) {
	sort.Slice(stockSlice, func(i, j int) bool {
		return stockSlice[i].Title < stockSlice[j].Title
	})
}

func print2Cols(stockSlice []*stocks.Stock, term *tty.Tty) {

	term.EraseEntireLine()

	// left -> Left side of the Slice, excluding the first value
	// because it was already printed.
	// right -> Right side of the Slice
	left, right := stockSlice[:(len(stockSlice) / 2)], stockSlice[len(stockSlice) / 2:]

	var mostWideLine int = 0
	var strBuilder strings.Builder

	for _, stock := range left {
		fmt.Fprintf(&strBuilder, "%s\t\t%s", stock.Title, stock.Description)
		if strBuilder.Len() > mostWideLine { mostWideLine = strBuilder.Len() }
		strBuilder.Reset()
	}

	for i, stock := range left {
		term.MoveCurPos(term.Rows(), uint16(1))
		fmt.Printf("%s\t\t%s", stock.Title, stock.Description)

		term.MoveCurPos(term.Rows(), uint16(mostWideLine + 12))
		fmt.Printf("|\t\t%s\t\t%s\n", right[i].Title, right[i].Description)
	}

	fmt.Println()
}

func startTrackerNotifications(ctx context.Context, conf *config.Config, service *stocks.Service) (chan<- *stocks.Stock, *map[string]*stocks.Stock) {

	c := make(chan *stocks.Stock)
	tempMap := make(map[string]*stocks.Stock)

	if trackedStocks, err := service.List(ctx); err == nil && len(trackedStocks) > 0 {
		for _, v := range trackedStocks {
			tempMap[v.Title] = v
		}
	}	

	go func() {

		// It will translate to 0.02 because I won't be working
		// directly with floats.
		error_rate := 2

		// t := time.NewTicker(5 * time.Minute) // It will fetch the current value every 5 minutes
		t := time.NewTicker(10 * time.Second) // It will fetch the current value every 5 minutes
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				close(c)
				t.Stop()
				return
			case <-t.C: // 5 minutes have passed
				if len(tempMap) <= 0 {
					break
				}

				for _, stock := range tempMap {
					updatedStock, err := brapi.GetStock(ctx, *stock)

					if err != nil {
						fmt.Printf("ERROR: %v\n", err)
						break
					}

					current := int(updatedStock.StockVal * 100)

					for _, target := range stock.TargetVals {
						below_target, above_target := int(target * 100) - error_rate, int(target * 100) + error_rate

						if current >= below_target && current <= above_target {
							// Send notification here
							err = conf.StockNotifier.Notify(notifications.Notification{
								TimeToExpire: 5e3,
								StockTitle: stock.Title,
								Body: fmt.Sprintf("%s acabou de bater %.2f. Abra agora seu homebroaker.", stock.Title, target),
							})

							if err != nil {
								fmt.Printf("\n\n%v\n\n", err)
								continue
							}
						}
					}
				}
			case stock, ok := <-c:
				if !ok {
					fmt.Printf("ERROR: Couldn't assign.\n")
					return
				}

				if slices.Contains(slices.Collect(maps.Keys(tempMap)), stock.Title) {
					if newTargetVals := tempMap[stock.Title]; !slices.Contains(newTargetVals.TargetVals, stock.StockVal) {
						newTargetVals.TargetVals = append(newTargetVals.TargetVals, stock.StockVal)

						err := service.Update(ctx, newTargetVals.ID, newTargetVals.Title, newTargetVals.Description, 0, 0, newTargetVals.StockVal, newTargetVals.TargetVals)

						if err != nil {
							fmt.Printf("\n\nERROR: %v\n\n", err)
							panic(1)
						}
					}
					break
				}

				stock, err := service.Create(ctx, stock.Title, stock.Description, 0, 0, stock.StockVal, []float64{stock.StockVal})

				if err != nil {
					fmt.Printf("\n\nERROR: %v\n\n", err)
					break
					// panic(1)
				}

				tempMap[stock.Title] = stock
			}
		}
	}()

	return c, &tempMap
}

func treatTargetValue(s *stocks.Stock, target_val string) (*stocks.Stock, error) {
	var (
		stock = *s
		treated_string string = target_val
		percent_index int
		operator byte = 0
		auxTargetVal int = 0
		mantissa, exponent string
	)

	// Cut off the operator, which is the first digit (if present)
	if strings.ContainsAny(treated_string, "-+") {
		operator = treated_string[0]
		treated_string = treated_string[1:]
	}

	// Cut off the percentage digit, which is the last digit (if present)
	if percent_index = strings.Index(treated_string, "%"); percent_index != -1 {
		treated_string = treated_string[:percent_index]
	}

	mantissa, exponent, found := strings.Cut(treated_string, `.`) 

	for i, r := 0, len(mantissa) - 1; i <= r; i++ {
		auxTargetVal += int(mantissa[i] - 48) * int(math.Pow10(r - i))
	}

	auxTargetVal = auxTargetVal * 100

	if found != false {
		exponent_bytes := bytes.Repeat([]byte("0"), 2)
		copy(exponent_bytes, exponent)
		for i, r := 1, len(exponent_bytes); i <= r; i++ {
			auxTargetVal += int(exponent_bytes[i - 1] - 48) * int(math.Pow10(r - i))
		}
	}

	if percent_index != -1 {
		if operator == '+' {
			stock.StockVal += float64(int(stock.StockVal * 1e2) * auxTargetVal) / 1e6
		} else if operator == '-' {
			if auxTargetVal >= 1e4 { 
				return &stocks.Stock{}, fmt.Errorf("ERROR: Can't assign a value below 0.\n")
			}
			stock.StockVal -= float64(int(stock.StockVal * 1e2) * auxTargetVal) / 1e6
		} else {
			stock.StockVal = float64(int(stock.StockVal * 1e2) * auxTargetVal) / 1e6
		}
		return s, nil
	}

	if operator == '+' {
		stock.StockVal = float64(int(stock.StockVal * 1e2) + auxTargetVal) / 1e2
	} else if operator == '-' {
		if auxTargetVal > int(stock.StockVal * 1e2) {
			return &stocks.Stock{}, fmt.Errorf("ERROR: Can't assign a value below 0.\n")
		}
		stock.StockVal = float64(int(stock.StockVal * 1e2) - auxTargetVal) / 1e2
	} else {
		stock.StockVal = float64(auxTargetVal) / 1e2
	}

	return &stock, nil
}

func updateStocksRoutine(baseCtx context.Context, stock_hmap *stocks.Stocks) chan bool {
	c := make(chan bool)
	scheduledUpdate := time.NewTimer(time.Minute * 5)
	// scheduledUpdate := time.NewTimer(time.Second * 5)

	var wg sync.WaitGroup

	go func() {
		for {
			select {
			case <-c: 
				if !scheduledUpdate.Stop() {
					wg.Wait()
					c <- true
					break
				}

				ctx, stop := signal.NotifyContext(baseCtx, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGQUIT)
				stockChan := brapi.GetAllStocks(ctx)

				CLoop:
					for {
						select {
						case <-ctx.Done():
							fmt.Printf("\x1b[2K\r")
							stop()
							break CLoop
						case stockSlice := <-stockChan:
							if len(stockSlice) == 0 {
								fmt.Printf("\n\nERROR 0\n\n")
								break CLoop
							}
							stock_hmap.UpdateAll(stockSlice)
							break CLoop
						}
					}

				stop()
				scheduledUpdate.Reset(time.Second * 5)
				c <- true
			case <-scheduledUpdate.C:
				wg.Add(1)

				go func() {
					ctx, stop := signal.NotifyContext(baseCtx, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGQUIT)
					stockChan := brapi.GetAllStocks(ctx)

					ScheduledChanLoop:
						for {
							select {
							case <-ctx.Done():
								break ScheduledChanLoop
							case stockSlice := <-stockChan:
								if len(stockSlice) == 0 {
									fmt.Printf("\n\nERROR 0\n\n")
									break ScheduledChanLoop
								}
								stock_hmap.UpdateAll(stockSlice)
								break ScheduledChanLoop
							}
						}

					stop()
					scheduledUpdate.Reset(time.Second * 5)
					wg.Done()
				}()
			}
		}
	}()

	return c
}

func showStock(ctx context.Context, stockTitle string, s *stocks.Stocks, conf *config.Config, term *tty.Tty) error {
	// Sadly, and unknowingly, Brapi doesn't show a lot of data for stocks
	// which are fractional (has an "F" in it's identifier). Also, the URL
	// for the company logo of fractional stocks are default pointing to
	// the Brapi'slogo. Because of all that, this function takes off
	// the "F" and then proceed to show the info about it.

	stockTitle = strings.Replace(strings.ToUpper(stockTitle), "F", "", -1)

	stock, err := s.Search(stockTitle)

	if err != nil {
		return err
	}

	stock.String()

	tempStock, err := brapi.GetStock(ctx, *stock)

	if err != nil {
		return err
	}

	stock.StockVal = tempStock.StockVal

	img_bytes, err := brapi.GetStockCompanyLogo(ctx, *stock)

	if err != nil {
		return err
	}

	str := strings.Join([]string{conf.TrackedAssetsLogos, stock.Title, `.svg`}, "")
	img_file_svg, err := os.Create(str)

	if err != nil {
		return err
	}

	_, err = img_file_svg.Write(img_bytes)

	if err != nil {
		return err
	}

	img_file_png := strings.Join([]string{conf.TrackedAssetsLogos, stock.Title, `.png`}, "")

	cmd := exec.Command("/usr/bin/ffmpeg", "-loglevel", "quiet", "-i", img_file_svg.Name(), img_file_png)

	err = cmd.Start()

	if err != nil {
		return err
	}

	err = cmd.Wait()

	if err != nil {
		return err
	}

	old_termios := term.GetTermios()

	term.SttySane()

	cmd = exec.Command("/usr/bin/chafa", "-s", "35x35", img_file_png)

	cmd.Stdout = os.Stdout

	err = cmd.Start()

	if err != nil {
		return err
	}

	err = cmd.Wait()

	if err != nil {
		return err
	}

	err = term.SetTermios(&old_termios)

	if err != nil {
		return err
	}

	return nil
}
