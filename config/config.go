package config

import (
	"bufio"
	"context"
	"fmt"
	"github.com/Laps14/assets-tracker/tty"
	"github.com/Laps14/assets-tracker/notifications"
	"os"
	"os/exec"
	"strings"
)

var (
	RIGHT_MEDIUM_TRIANGLE = '\u23F5'
	UP_MEDIUM_TRIANGLE = '\u23F6'
	DOWN_MEDIUM_TRIANGLE = '\u23F6'
	UP_DOUBLE_TRIANGLE = '\u23EB'
	DOWN_DOUBLE_TRIANGLE = '\u23EC'
)

const (
	REGULAR_USER_PERM = 0o0744
)

type Config struct {
	UserHomeDir string
	TrackerConfDir string
	TrackedAssetsLogos string
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
	c.TrackedAssetsLogos = os.TempDir() + "/assets-tracker/logos/"
	c.ClientConf = c.TrackerConfDir + "client.conf"
	c.selectDisplayServer()

	cmd := exec.Command("whoami")

	// Checks for user name to set it as database admin name and password
	userName, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("Couldn't get your username")
		panic(1)
	}

	err = os.Setenv("POSTGRES_USER", strings.TrimSpace(string(userName)))

	if err != nil {
		fmt.Println("Couldn't get your username")
		panic(1)
	}

	os.Setenv("POSTGRES_PASSWORD", strings.TrimSpace(string(userName)))

	if err != nil {
		fmt.Println("Couldn't get your username")
		panic(1)
	}

	c.startDatabaseContainer()

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

	tracked_assets_logos, err := os.Open(c.TrackedAssetsLogos)

	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(c.TrackedAssetsLogos, os.ModeDir | REGULAR_USER_PERM)

			if err != nil {
				fmt.Printf("MKDIR\n%v\n", err)
				panic(1)
			}

			tracked_assets_logos, err = os.Open(c.TrackedAssetsLogos)

			if err != nil {
				fmt.Printf("\n%v\n", err)
				panic(1)
			}
		}
	}
	defer tracked_assets_logos.Close()

	// Checks if the client.conf File exists. If not, creates it
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

	n, err = fmt.Fprintf(f, "ACCESS_TOKEN=%s\n", access_token)

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
	case "hyprland":
		c.StockNotifier = notifications.NewHyprlandNotifier()
	}

	return nil
}

func (c *Config) startDatabaseContainer() error {

	fmt.Println("Para a criação/inicialização do Banco de Dados do tracker será necessário que você forneça privilégios de administrador.\n")
	cmd := exec.Command("sudo", "docker", "image", "ls", "--format", `{{.Repository}}`)

	dockerImages, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("\n\n%v\n\n\n\n%v\n\n", dockerImages, err)
		return err
	}

	if !strings.Contains(string(dockerImages), "assets-tracker-postgres") {

		pg_user := "POSTGRES_USER=" + os.Getenv("POSTGRES_USER")
		pg_pwd := "POSTGRES_PASSWORD=" + os.Getenv("POSTGRES_PASSWORD")
		pg_defaultDB := "POSTGRES_DB=assets_tracker"

		cmd = exec.Command("sudo", "docker", "buildx", "build", "-t","assets-tracker-postgres", "-f", c.UserHomeDir + "/assets-tracker/Dockerfile", `.`)

		err = cmd.Run()

		if err != nil {
			fmt.Printf("Failed to create the database image: %v\n", err)
			panic(1)
		}

		cmd = exec.Command("sudo", "docker", "run", "-d", "-p", "127.0.0.1:1377:5432/tcp", "--name", "assets-tracker-database", "-e", pg_pwd, "-e", pg_user, "-e", pg_defaultDB, "assets-tracker-postgres")

		err = cmd.Run()

		if err != nil {
			fmt.Printf("Failed to create the database container: %v\n", err)
			panic(1)
		}
	} else {
		cmd = exec.Command("sudo", "docker", "start", "assets-tracker-database")

		err = cmd.Run()

		if err != nil {
			panic(1)
		}
	}

	fmt.Println("Banco de Dados iniciado com sucesso na porta 1377\n")
	return nil
}
