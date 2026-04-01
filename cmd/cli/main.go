package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"github.com/Laps14/assets-tracker/config"
	"github.com/Laps14/assets-tracker/internal/http/brapi"
	"github.com/Laps14/assets-tracker/tty"
	"golang.org/x/sys/unix"
	"strings"
	"text/template"
)

const (
	help_templ = "\n\x1b[1mh, help\x1b[0m\n\t Apresenta esta mensagem de ajuda.\n\n" +
		"\x1b[1ml, list\x1b[0m\n\t Lista todas as ações possíveis.\n\n" +
		"\x1b[1m--tracked\x1b[0m\n\t Opção usada com list para listar apenas as ações monitoradas.\n\n" +
		"\x1b[1mw, write STOCK [STOCK...]\x1b[0m\n\t Salva a(s) STOCK(s) para serem monitoradas.\n\n" +
		"\x1b[1mSTOCK\x1b[0m\n\t O código/sigla de uma ação. Caso o tipo (3, 4, 5, 3F etc.) seja omitido, por padrão, a ação com indicador 4F será utilizada.\n\n"
)

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
	mainSignals := make(chan os.Signal)

	term.ShutdownTtyRoutine(mainSignals) // Shutdown goroutine

	conf.CheckDirectories(ctx, term)

	ctx = brapi.NewContext(ctx, conf.AccessToken)

	brapi.NewBrapiClient(ctx)

	// Every command calls signal.Reset because it will be able
	// to cancel itself without quitting the tracker.
	// After the execution of a command, the main will be able
	// to receive signals again.

	for {
		getCmd(ctx)
		signal.Notify(mainSignals, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGQUIT)
	}
}

func getCmd(ctx context.Context) {
	fmt.Print("\x1b[1m\u276F\u276F \x1b[0m")

	buf := bufio.NewReader(os.Stdin)

	line, err := buf.ReadString('\n')

	if err != nil {
		fmt.Errorf("Error while reading the input")
		panic(1)
	}

	args := strings.Fields(line)

	switch args[0] {
	case "w", "write":
		return
	case "l", "list":
		c := brapi.GetStocks(ctx)

		for i := range c {
			fmt.Println(i)
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
