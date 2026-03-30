package config

import (
	"bufio"
	"context"
	// "errors"
	"fmt"
	"github.com/Laps14/assets-tracker/tty"
	"os"
	"strings"
	// "io"
)

var (
	RIGHT_MEDIUM_TRIANGLE = '\u23F5'
	UP_MEDIUM_TRIANGLE = '\u23F6'
	DOWN_MEDIUM_TRIANGLE = '\u23F6'
	UP_DOUBLE_TRIANGLE = '\u23EB'
	DOWN_DOUBLE_TRIANGLE = '\u23EC'
)

const REGULAR_USER_PERM = 0o0744

type Config struct {
	UserHomeDir string
	TrackerConfDir string
	TrackedAssets string
	ClientConf string
}

func InitializeConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		panic(1)
	}

	c := &Config{
		UserHomeDir: homeDir,
	}

	c.TrackerConfDir = c.UserHomeDir + "/.assets-tracker/"
	c.TrackedAssets = c.TrackerConfDir + "tracked-assets.conf"
	c.ClientConf = c.TrackerConfDir + "client.conf"

	return c, nil
}

func (c *Config) CheckDirectories(ctx context.Context, term *tty.Tty) {

	tracker_conf_dir, err := os.Open(c.TrackerConfDir)

	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(c.TrackerConfDir, os.ModeDir | REGULAR_USER_PERM)
			tracker_conf_dir, err = os.Open(c.TrackerConfDir)

			if err != nil {
				panic(1)
			}
		}
	}
	defer tracker_conf_dir.Close()

	tracked_assets, err := os.Open(c.TrackedAssets)

	if err != nil {
		if os.IsNotExist(err) {
			tracked_assets, err = os.Create(c.TrackedAssets)

			if err != nil {
				panic(1)
			}

			tracked_assets.Chmod(REGULAR_USER_PERM)
		}
	}
	defer tracked_assets.Close()

	client_conf, err := os.Open(c.ClientConf)

	if err != nil {
		if os.IsNotExist(err) {
			client_conf, err = os.Create(c.ClientConf)

			if err != nil {
				panic(1)
			}

			client_conf.Chmod(REGULAR_USER_PERM)

			c.askForAccessToken(ctx, client_conf, term)
		}
	}
	defer client_conf.Close()

	conf_reader := bufio.NewScanner(client_conf)

	for {
		if !conf_reader.Scan() { break }

		config_line := conf_reader.Text()

		key_val := strings.Split(config_line, "=")

		if key_val[0] == "ACCESS_TOKEN" { break }
	}
}

func (c *Config) askForAccessToken(ctx context.Context, f *os.File, term *tty.Tty) {
	go func() {
		<-ctx.Done()
		term.EnableEcho()
		term.ShutdownTtyPrompt(ctx)
	}()

	var access_token string

	fmt.Println("É necessário informar o seu token de autenticação para a brapi.\nCaso ainda não tenha um token, acesse https://brapi.dev/dashboard para obter um.\n")
	fmt.Print("Informe seu token (por segurança, o token digitado não será mostrado em tela): ")

	term.DisableEcho()
	n, err := fmt.Scanf("%s", &access_token)
	term.EnableEcho()
	fmt.Println()

	if err != nil {
		fmt.Printf("- Não foi possível registrar seu token de acesso. Tente novamente.\n")
		c.askForAccessToken(ctx, f, term)
		return
	}

	config_line := fmt.Sprintf("ACCESS_TOKEN=%s\n", access_token)
	n, err = f.Write([]byte(config_line))

	if err != nil || n == 0 {
		fmt.Printf("- Não foi possível registrar seu token de acesso. Tente novamente.\n")
		c.askForAccessToken(ctx, f, term)
		return
	}
}
