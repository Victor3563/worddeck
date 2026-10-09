package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/worddeck/internal/logger"
)

const defaultPort = 8080

type Config struct {
	Port              int
	DatabaseURL       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxIdleTime time.Duration
	DBConnMaxLifetime time.Duration
	SessionTTL        time.Duration
	// false — только для локального запуска по HTTP.
	CookieSecure          bool
	LogLevel              slog.Level
	LogFormat             string // json | text
	ShutdownTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPReadTimeout       time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	MaxImportBytes        int64
	DBConnectTimeout      time.Duration
}

// Настройки, нужные только серверу, не проверяются, чтобы migrate запускался
// с минимальным окружением; перед serve вызывать ValidateServe.
func Load() (Config, error) {
	var errs []error
	c := Config{
		Port:                  intEnv("PORT", defaultPort, &errs),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		DBMaxOpenConns:        intEnv("DB_MAX_OPEN_CONNS", 10, &errs),
		DBMaxIdleConns:        intEnv("DB_MAX_IDLE_CONNS", 5, &errs),
		DBConnMaxIdleTime:     durationEnv("DB_CONN_MAX_IDLE_TIME", 5*time.Minute, &errs),
		DBConnMaxLifetime:     durationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute, &errs),
		SessionTTL:            durationEnv("SESSION_TTL", 30*24*time.Hour, &errs),
		CookieSecure:          boolEnv("COOKIE_SECURE", true, &errs),
		LogFormat:             strings.ToLower(strEnv("LOG_FORMAT", logger.FormatJSON)),
		ShutdownTimeout:       durationEnv("SHUTDOWN_TIMEOUT", 15*time.Second, &errs),
		HTTPReadHeaderTimeout: durationEnv("HTTP_READ_HEADER_TIMEOUT", 5*time.Second, &errs),
		HTTPReadTimeout:       durationEnv("HTTP_READ_TIMEOUT", 30*time.Second, &errs),
		HTTPWriteTimeout:      durationEnv("HTTP_WRITE_TIMEOUT", 60*time.Second, &errs),
		HTTPIdleTimeout:       durationEnv("HTTP_IDLE_TIMEOUT", 120*time.Second, &errs),
		MaxImportBytes:        int64(intEnv("MAX_IMPORT_BYTES", 1<<20, &errs)),
		DBConnectTimeout:      durationEnv("DB_CONNECT_TIMEOUT", 30*time.Second, &errs),
	}

	if err := c.LogLevel.UnmarshalText([]byte(strEnv("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}
	if !logger.ValidFormat(c.LogFormat) {
		errs = append(errs, fmt.Errorf("LOG_FORMAT must be json or text, got %q", c.LogFormat))
	}
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.Port <= 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("PORT out of range: %d", c.Port))
	}
	return c, errors.Join(errs...)
}

func (c Config) ValidateServe() error {
	var errs []error
	if c.SessionTTL <= 0 {
		errs = append(errs, errors.New("SESSION_TTL must be positive"))
	}
	if c.MaxImportBytes <= 0 {
		errs = append(errs, errors.New("MAX_IMPORT_BYTES must be positive"))
	}
	return errors.Join(errs...)
}

func strEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func intEnv(key string, def int, errs *[]error) int {
	v := strEnv(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
		return def
	}
	return n
}

func boolEnv(key string, def bool, errs *[]error) bool {
	v := strEnv(key, "")
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
		return def
	}
	return b
}

func durationEnv(key string, def time.Duration, errs *[]error) time.Duration {
	v := strEnv(key, "")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
		return def
	}
	return d
}
