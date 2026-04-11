package config

import (
	"bufio"
	"context"
	// "errors"
	"fmt"
	"github.com/Laps14/assets-tracker/tty"
	"github.com/Laps14/assets-tracker/notifications"
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
	AccessToken string
	XDGSessionDesktop string
	StockNotifier notifications.Notifier
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
	c.selectDisplayServer()

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

	// Checks if the client.conf file exists. If not, creates it
	// and ask for the user access token to save it and use
	// for requests later.
	
	client_conf, err := os.Open(c.ClientConf)

	if err != nil {
		if os.IsNotExist(err) {
			client_conf, err = os.Create(c.ClientConf)

			if err != nil {
				panic(1)
			}

			client_conf.Chmod(REGULAR_USER_PERM)

			c.askForAccessToken(ctx, client_conf, term)
			return
		}
	}
	defer client_conf.Close()

	// If client.conf already exists, traverse through it
	// and assigns the save token to the Config struct

	conf_reader := bufio.NewScanner(client_conf)

	for {
		if !conf_reader.Scan() {
			if len(c.AccessToken) == 0 {
				c.askForAccessToken(ctx, client_conf, term)
			}
		}

		config_line := conf_reader.Text()

		key_val := strings.Split(config_line, "=")

		if key_val[0] == "ACCESS_TOKEN" {
			if len(key_val[1]) > 0 {
				c.AccessToken = key_val[1]
				return
			}
		}
	}
}

func (c *Config) askForAccessToken(ctx context.Context, f *os.File, term *tty.Tty) {
	go func() {
		<-ctx.Done()
		term.EnableEcho()
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

	c.AccessToken = access_token
}

func (c *Config) selectDisplayServer() error {
	val, ok := os.LookupEnv("XDG_SESSION_DESKTOP")
	if !ok {
		return fmt.Errorf("ERROR: XDG_SESSION_DESKTOP not set.\n")
	}

	if len(val) == 0 {
		return fmt.Errorf("ERROR: There's no active desktop environment or compositor active.\n")
	}

	c.XDGSessionDesktop = val

	switch strings.ToLower(val) {
	case "cinnamon":
		c.StockNotifier = notifications.NewLibNotifyNotifier()
	}

	return nil
}
