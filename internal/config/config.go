package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

// Parse 解析命令行参数与环境变量（输出到 stderr）。
// 参数解析失败返回错误（flag.ErrHelp 表示用户请求了帮助）。
func Parse() (*Config, error) {
	return parseArgs(os.Args[1:], os.Stderr)
}

// ParseArgs 从指定参数与输出流解析配置，便于测试注入。
func ParseArgs(args []string, output io.Writer) (*Config, error) {
	return parseArgs(args, output)
}

// parseArgs 从指定参数与输出流解析配置，便于测试注入。
func parseArgs(args []string, output io.Writer) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("boxd", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.Usage = func() {
		printUsage(fs)
	}
	fs.StringVar(&cfg.Listen, "listen", resolveListen(), "listen address")
	fs.StringVar(&cfg.ConfigPath, "config", getEnv("BOXD_CONFIG", "/etc/sing-box/config.json"), "sing-box config path")
	fs.StringVar(&cfg.DataDir, "data-dir", getEnv("BOXD_DATA_DIR", "/var/lib/boxd"), "data directory")
	fs.StringVar(&cfg.Username, "username", getEnv("BOXD_USERNAME", "admin"), "login username")
	fs.StringVar(&cfg.Password, "password", getEnv("BOXD_PASSWORD", ""), "login password")
	fs.IntVar(&cfg.RefreshInterval, "refresh-interval", getEnvInt("BOXD_REFRESH_INTERVAL", 60), "subscription refresh interval (minutes)")
	fs.StringVar(&cfg.TLSCert, "tls-cert", getEnv("BOXD_TLS_CERT", ""), "TLS certificate file path")
	fs.StringVar(&cfg.TLSKey, "tls-key", getEnv("BOXD_TLS_KEY", ""), "TLS private key file path")
	fs.StringVar(&cfg.LogLevel, "log-level", getEnv("BOXD_LOG_LEVEL", "info"), "log level (debug|info|warn|error)")
	fs.StringVar(&cfg.BackupPath, "backup", "", "create a backup archive and exit")
	fs.StringVar(&cfg.RestorePath, "restore", "", "restore a backup archive and exit")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.CORSAllowedOrigins = parseCORSOrigins(os.Getenv("BOXD_CORS_ALLOWED_ORIGINS"))

	return cfg, nil
}

// printUsage 输出 boxd 命令用法。
func printUsage(fs *flag.FlagSet) {
	_, _ = fmt.Fprintf(fs.Output(), "Usage: boxd [options]\n\n")
	_, _ = fmt.Fprintf(fs.Output(), "boxd — single-node control plane for sing-box\n\nOptions:\n")
	fs.PrintDefaults()
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
// 优先级：--listen / BOXD_LISTEN（完整地址）> BOXD_PORT（仅端口号）> 默认 [::]:9091
func resolveListen() string {
	if addr := getEnv("BOXD_LISTEN", ""); addr != "" {
		return addr
	}
	if port := getEnv("BOXD_PORT", ""); port != "" {
		return "[::]:" + port
	}
	return "[::]:9091"
}

// IsHelpError 判断是否因请求帮助（--help）返回。
func IsHelpError(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}
