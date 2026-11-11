package runtime

import (
	"os"
	"path/filepath"
)

const (
	modeEmbedded = "embedded"
	modeRemote   = "remote"
)

type Config struct {
	Mode            string
	RemoteURL       string
	DataDir         string
	ConfigPath      string
	Username        string
	Password        string
	RefreshInterval int
	SkipAutostart   bool
}

func DefaultConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	boxdDir := filepath.Join(home, ".boxd")
	return Config{
		Mode:            modeEmbedded,
		RemoteURL:       "http://127.0.0.1:9091",
		DataDir:         boxdDir,
		ConfigPath:      filepath.Join(boxdDir, "config.json"),
		Username:        "admin",
		RefreshInterval: 60,
	}
}

func LoadConfig() (Config, error) {
	return loadConfig(os.Getenv, os.UserHomeDir)
}

func loadConfig(getenv func(string) string, userHomeDir func() (string, error)) (Config, error) {
	cfg := Config{
		Mode:            modeEmbedded,
		RemoteURL:       "http://127.0.0.1:9091",
		Username:        "admin",
		RefreshInterval: 60,
	}
	home, err := userHomeDir()
	if err != nil {
		home = "."
	}
	boxdDir := filepath.Join(home, ".boxd")
	cfg.DataDir = boxdDir
	cfg.ConfigPath = filepath.Join(boxdDir, "config.json")
	if value := getenv("BOXD_DESKTOP_MODE"); value == modeRemote {
		cfg.Mode = modeRemote
	}
	if value := getenv("BOXD_REMOTE_URL"); value != "" {
		cfg.RemoteURL = value
	}
	if value := getenv("BOXD_DATA_DIR"); value != "" {
		cfg.DataDir = value
	}
	if value := getenv("BOXD_CONFIG"); value != "" {
		cfg.ConfigPath = value
	}
	if value := getenv("BOXD_USERNAME"); value != "" {
		cfg.Username = value
	}
	if value := getenv("BOXD_PASSWORD"); value != "" {
		cfg.Password = value
	}
	if getenv("BOXD_SKIP_KERNEL_AUTOSTART") == "1" {
		cfg.SkipAutostart = true
	}
	return cfg, nil
}
