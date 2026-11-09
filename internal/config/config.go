// Package config manages runtime operational parameters and infrastructure configurations.
package config

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"golingko/pkg/database"
	"golingko/pkg/env"
)

type DatabaseConfig = database.Config

// Root operational settings.
type Config struct {
	ServerAddr string
	HTTPRouter string // Multiplexer driver used for HTTP routing (e.g. "nethttp", "chi", "gin").

	DB DatabaseConfig

	RedisURL string

	MinTapBalance    int64 // in IDR (Rp)
	JakLingkoFareCap int64 // in IDR (Rp)
	// Duration before an unclosed journey leg is deemed abandoned.
	JourneyExpiryHours time.Duration

	Logging LoggingConfig
}

type LoggingConfig struct {
	Driver    string // Driver engine to instantiate.
	Verbosity string // Minimum emission threshold for records
	// Layout format for record.
	Format string
}

// Populates runtime configuration from environment variables.
func Load() *Config {
	env.Load(".env")

	dbCfg := DatabaseConfig{
		URL:      env.Get("DATABASE_URL", ""),
		Host:     env.Get("DB_HOST", "localhost"),
		Port:     env.Get("DB_PORT", 5432),
		User:     env.Get("DB_USER", "golingko_user"),
		Password: env.Get("DB_PASSWORD", "golingko_password"),
		DBName:   env.Get("DB_NAME", "golingko_db"),
		SSLMode:  env.Get("DB_SSLMODE", "disable"),
	}

	if dbCfg.URL != "" {
		if u, err := url.Parse(dbCfg.URL); err == nil {
			if h := u.Hostname(); h != "" {
				dbCfg.Host = h
			}
			if p := u.Port(); p != "" {
				if portNum, err := strconv.Atoi(p); err == nil {
					dbCfg.Port = portNum
				}
			}
			if u.User != nil {
				dbCfg.User = u.User.Username()
				if pass, ok := u.User.Password(); ok {
					dbCfg.Password = pass
				}
			}
			if path := strings.TrimPrefix(u.Path, "/"); path != "" {
				dbCfg.DBName = path
			}
			if qSSL := u.Query().Get("sslmode"); qSSL != "" {
				dbCfg.SSLMode = qSSL
			}
		}
	}

	serverAddr := env.Get("SERVER_ADDR", env.Get("ADDR", ""))
	if serverAddr == "" {
		port := env.Get("PORT", env.Get("SERVER_PORT", "8080"))
		if !strings.Contains(port, ":") {
			serverAddr = "127.0.0.1:" + port
		} else {
			serverAddr = port
		}
	}

	return &Config{
		ServerAddr:         serverAddr,
		HTTPRouter:         env.Get("HTTP_ROUTER", env.Get("ROUTER_ENGINE", "nethttp")),
		DB:                 dbCfg,
		RedisURL:           env.Get("REDIS_URL", "localhost:6379"),
		MinTapBalance:      env.Get("MIN_TAP_BALANCE", int64(5000)),
		JakLingkoFareCap:   env.Get("JAKLINGKO_FARE_CAP", int64(10000)),
		JourneyExpiryHours: time.Duration(env.Get("JOURNEY_EXPIRY_HOURS", 4)) * time.Hour,
		Logging: LoggingConfig{
			Driver:    env.Get("LOG_DRIVER", "slog"),
			Verbosity: env.Get("LOG_VERBOSITY", env.Get("LOG_LEVEL", "info")),
			Format:    env.Get("LOG_FORMAT", "json"),
		},
	}
}

func loadEnvFile(path string) {
	env.Load(path)
}
