package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Listen             string
	ConfigPath         string
	DataDir            string
	Username           string
	Password           string
	RefreshInterval    int
	CORSAllowedOrigins []string
	TLSCert            string
	TLSKey             string
	LogLevel           string
	BackupPath         string
	RestorePath        string
	ShowVersion        bool
}

func Parse() *Config {
	cfg := &Config{}

	fs := flag.NewFlagSet("boxui", flag.ContinueOnError)
	fs.StringVar(&cfg.Listen, "listen", resolveListen(), "listen address")
	fs.StringVar(&cfg.ConfigPath, "config", getEnv("BOXUI_CONFIG", "/etc/sing-box/config.json"), "sing-box config path")
	fs.StringVar(&cfg.DataDir, "data-dir", getEnv("BOXUI_DATA_DIR", "/var/lib/boxui"), "data directory")
	fs.StringVar(&cfg.Username, "username", getEnv("BOXUI_USERNAME", "admin"), "login username")
	fs.StringVar(&cfg.Password, "password", getEnv("BOXUI_PASSWORD", ""), "login password")
	fs.IntVar(&cfg.RefreshInterval, "refresh-interval", getEnvInt("BOXUI_REFRESH_INTERVAL", 60), "subscription refresh interval (minutes)")
	fs.StringVar(&cfg.TLSCert, "tls-cert", getEnv("BOXUI_TLS_CERT", ""), "TLS certificate file path")
	fs.StringVar(&cfg.TLSKey, "tls-key", getEnv("BOXUI_TLS_KEY", ""), "TLS private key file path")
	fs.StringVar(&cfg.LogLevel, "log-level", getEnv("BOXUI_LOG_LEVEL", "info"), "log level (debug|info|warn|error)")
	fs.StringVar(&cfg.BackupPath, "backup", "", "create a backup archive and exit")
	fs.StringVar(&cfg.RestorePath, "restore", "", "restore a backup archive and exit")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")
	_ = fs.Parse(os.Args[1:])

	cfg.CORSAllowedOrigins = parseCORSOrigins(os.Getenv("BOXUI_CORS_ALLOWED_ORIGINS"))

	return cfg
}

func parseCORSOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvList(key string) []string {
	return parseCORSOrigins(os.Getenv(key))
}

// resolveListen 解析最终监听地址。
// 优先级：--listen / BOXUI_LISTEN（完整地址）> BOXUI_PORT（仅端口号）> 默认 [::]:9091
func resolveListen() string {
	if addr := getEnv("BOXUI_LISTEN", ""); addr != "" {
		return addr
	}
	if port := getEnv("BOXUI_PORT", ""); port != "" {
		return "[::]:" + port
	}
	return "[::]:9091"
}
