package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/Laps14/assets-tracker/config"
	"github.com/Laps14/assets-tracker/internal/http/brapi"
	"github.com/Laps14/assets-tracker/notifications"
	"github.com/Laps14/assets-tracker/stocks"
	"github.com/Laps14/assets-tracker/tty"
	"golang.org/x/sys/unix"
	"maps"
	"math"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"sort"
	"strings"
	"text/template"
	"time"
)

const (
	help_templ = "\n\x1b[1mh, help\x1b[0m\n\t Apresenta esta mensagem de ajuda.\n\n" +
		"\x1b[1ml, list\x1b[0m\n\t Lista todas as ações possíveis.\n\n" +
		"\x1b[1m--tracked\x1b[0m\n\t Opção usada com list para listar apenas as ações monitoradas.\n\n" +
		"\x1b[1mw, write STOCK [STOCK...]\x1b[0m\n\t Salva a(s) STOCK(s) para serem monitoradas. Para mais de uma ação, separá-las por espaço.\n\n" +
		"\x1b[1mSTOCK\x1b[0m\n\t O código/sigla de uma ação. Caso o tipo (3, 4, 5, 3F etc.) seja omitido, por padrão, a ação com indicador 4F será utilizada.\n\n"
)

var loading_chars = [...]string{"\u2819", "\u2838", "\u28B0", "\u28E0", "\u28C4", "\u2846", "\u2807"}

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

	term.ShutdownTtyRoutine(mainSignals) // Shutdown goroutine

	conf.CheckDirectories(ctx, term)

	ctx = brapi.NewContext(ctx, conf.AccessToken)

	brapi.NewBrapiClient(ctx)

	stock_hmap := stocks.NewStocks()

	trackedStocks := make(map[string]stocks.Stock)

	trackedStocksChan := startTrackerNotifications(ctx, &trackedStocks, conf)

	// Some commands call signal.Reset because it will be able
	// to cancel "itself" without quitting the tracker.
	// After the execution of a command, the main will be able
	// to receive signals again.

	for {
		getCmd(ctx, term, stock_hmap, trackedStocksChan)
		signal.Notify(mainSignals, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGQUIT)
	}
}

func getCmd(ctx context.Context, term *tty.Tty, s *stocks.Stocks, trackedStocksChan chan<- stocks.Stock) {
	fmt.Print("\x1b[1m\u276F\u276F \x1b[0m")

	buf := bufio.NewReader(os.Stdin)

	line, err := buf.ReadString('\n')

	if err != nil {
		fmt.Errorf("Error while reading the input")
		panic(1)
	} else if len(line) <= 1 {
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

		// reg := regexp.MustCompile(`\b\w+\d{0,2}\w?\b[[:blank:]]\b\d+\b`)
		// reg := regexp.MustCompile(`\b[A-Z]{4}\d*[F]?[[:blank:]]\d+[\d{1,2}|\%]?\b`)
		reg := regexp.MustCompile(`\b[A-Z]{4}\d*[F]?[[:blank:]][\-\+]?\d+(?:\.\d{1,2})?(?:\%\B)?`)

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

		for i, r := 0, args[1:]; i < len(r); i += 2 {
			stock, err := s.Search(r[i])

			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
				continue
			}

			stockWithTargetVal, err := treatTargetValue(*stock, r[i+1])

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
				fmt.Println("--tracked")
				return
			} else {
				var str string
				str = strings.Join(args[1:], " ")

				// Checks if the input line has any non-word char
				// It can't be a simple \W because then the spaces
				// would be interpreted as errors.
				reg := regexp.MustCompile(`\b(?:[A-Z]{4}\d{0,2}[F]?)\s?`) 

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

		var ith_char int = 0

		ctx, stockchan := brapi.GetAllStocks(ctx)

		term.DisableCursor()
		defer term.EnableCursor()

		for{
			select {
			case <-ctx.Done():
				fmt.Printf("\x1b[2K\r")
				return
			case stockSlice, ok := <-stockchan:
				if !ok {
					fmt.Printf("\x1b[2K\r")
					return
				}
				s.InsertAll(stockSlice)
				sortByStock(stockSlice)
				print2Cols(stockSlice, term)
				return 
			default:
				loadingLine(&ith_char, term)
			}
		}

		return
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
	term.MoveCurPos(term.Rows(), 1)
	*ith_char++
}

func sortByStock(stockSlice []stocks.Stock) {
	sort.Slice(stockSlice, func(i, j int) bool {
		return stockSlice[i].Title < stockSlice[j].Title
	})
}

func print2Cols(stockSlice []stocks.Stock, term *tty.Tty) {

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

func startTrackerNotifications(ctx context.Context, trackedStocks *map[string]stocks.Stock, conf *config.Config) chan<- stocks.Stock {
	c := make(chan stocks.Stock)
	tempMap := *trackedStocks

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
				return
			case <-t.C: // 5 minutes have passed
				if len(tempMap) <= 0 {
					continue
				}

				// receives slice of Stock returns []stocks.Stock
				updatedStocks, err := brapi.GetStocks(ctx, slices.Collect(maps.Values(tempMap)))

				if err != nil {
					fmt.Printf("ERROR: %v\n", err)
					continue
				}

				for _, stock := range updatedStocks {
					current := int(stock.StockVal * 100)
					target := int(tempMap[stock.Title].StockVal * 100)
					below_target, above_target := target - error_rate, target + error_rate

					if current >= below_target && current <= above_target {
						// Send notification here
						err = conf.StockNotifier.Notify(notifications.Notification{
							TimeToExpire: 1e3,
							StockTitle: stock.Title,
							Body: fmt.Sprintf("%s acabou de bater %.2f. Abra agora seu homebroaker.", stock.Title, stock.StockVal),
						})

						if err != nil {
							fmt.Printf("\n\n%v\n\n", err)
							return
						}
					}
				}
			case stock, ok := <-c:
				if !ok {
					fmt.Printf("ERROR: Couldn't assign.\n")
					return
				}

				tempMap[stock.Title] = stock
				*trackedStocks = tempMap
			}
		}
	}()

	return c
}

func treatTargetValue(s stocks.Stock, target_val string) (stocks.Stock, error) {
	var (
		treated_string string = target_val
		percent_index int
		operator byte = 0
		auxTargetVal int = 0
		mantissa, exponent string
	)

	// Cut off the operator, which is the first digit
	if strings.ContainsAny(treated_string, "-+") {
		operator = treated_string[0]
		treated_string = treated_string[1:]
	}

	// Cut off the percentage char, which is the last digit (if present)
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
			s.StockVal += float64(int(s.StockVal * 1e2) * auxTargetVal) / 1e6
		} else if operator == '-' {
			if auxTargetVal >= 1e4 { 
				return stocks.Stock{}, fmt.Errorf("ERROR: Can't assign a value below 0.\n")
			}
			s.StockVal -= float64(int(s.StockVal * 1e2) * auxTargetVal) / 1e6
		} else {
			s.StockVal = float64(int(s.StockVal * 1e2) * auxTargetVal) / 1e6
		}
		return s, nil
	}

	if operator == '+' {
		s.StockVal = float64(int(s.StockVal * 1e2) + auxTargetVal) / 1e2
	} else if operator == '-' {
		if auxTargetVal > int(s.StockVal * 1e2) {
			return stocks.Stock{}, fmt.Errorf("ERROR: Can't assign a value below 0.\n")
		}
		s.StockVal = float64(int(s.StockVal * 1e2) - auxTargetVal) / 1e2
	} else {
		s.StockVal = float64(auxTargetVal) / 1e2
	}

	return s, nil
}
