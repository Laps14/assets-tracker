package config

import (
	"os"
)

var (
	RIGHT_MEDIUM_TRIANGLE = '\u23F5'
	UP_MEDIUM_TRIANGLE = '\u23F6'
	DOWN_MEDIUM_TRIANGLE = '\u23F6'
	UP_DOUBLE_TRIANGLE = '\u23EB'
	DOWN_DOUBLE_TRIANGLE = '\u23EC'
)

type Config struct {
	UserHomeDir string
	TrackerConfDir string
	TrackedAssets string
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

	return c, nil
}
