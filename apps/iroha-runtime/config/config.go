package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	EnvAniListUsername             = "IROHA_ANILIST_USERNAME"
	EnvAniListToken                = "IROHA_ANILIST_TOKEN"
	EnvAniListActivityLookbackDays = "IROHA_ANILIST_ACTIVITY_LOOKBACK_DAYS"
	EnvBangumiUsername             = "IROHA_BANGUMI_USERNAME"
	EnvBangumiToken                = "IROHA_BANGUMI_TOKEN"
	EnvTimezone                    = "IROHA_TIMEZONE"
	EnvPublicExportDir             = "IROHA_PUBLIC_EXPORT_DIR"
	EnvPublicExportPrivacy         = "IROHA_PUBLIC_EXPORT_PRIVACY"
)

// defaultAllowedOrigins lets the local web dev server reach the private API.
var defaultAllowedOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Storage  StorageConfig  `toml:"storage"`
	Cache    CacheConfig    `toml:"cache"`
}

type ServerConfig struct {
	Addr     string `toml:"addr"`
	Timezone string `toml:"timezone"`
	// AllowedOrigins restricts CORS for the private /api/v1 routes. The public
	// /public/v1 routes always allow all origins (sanitized data).
	AllowedOrigins []string `toml:"allowed_origins"`
}

type DatabaseConfig struct {
	URL string `toml:"url"`
}

type StorageConfig struct {
	DataDir string `toml:"data_dir"`
}

type CacheConfig struct {
	Backend string `toml:"backend"`
	URL     string `toml:"url"`
}

func Load(path string) (Config, error) {
	cfg := Default()
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return Config{}, err
		}
	} else if !os.IsNotExist(err) {
		return Config{}, err
	}

	applyEnv(&cfg)
	if cfg.Server.Timezone == "" {
		cfg.Server.Timezone = Default().Server.Timezone
	}
	if _, err := time.LoadLocation(cfg.Server.Timezone); err != nil {
		return Config{}, fmt.Errorf("invalid server timezone %q: %w", cfg.Server.Timezone, err)
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Addr:           "127.0.0.1:8080",
			Timezone:       "Asia/Tokyo",
			AllowedOrigins: defaultAllowedOrigins,
		},
		Database: DatabaseConfig{
			URL: "postgres://iroha:iroha_dev@localhost:5432/iroha?sslmode=disable",
		},
		Storage: StorageConfig{
			DataDir: ".iroha-data",
		},
		Cache: CacheConfig{
			Backend: "postgres",
		},
	}
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("IROHA_SERVER_ADDR"); value != "" {
		cfg.Server.Addr = value
	}
	if value := os.Getenv(EnvTimezone); value != "" {
		cfg.Server.Timezone = value
	}
	if value := os.Getenv("IROHA_DATABASE_URL"); value != "" {
		cfg.Database.URL = value
	}
	if value := os.Getenv("IROHA_DATA_DIR"); value != "" {
		cfg.Storage.DataDir = value
	}
	if value := os.Getenv("IROHA_VALKEY_URL"); value != "" {
		cfg.Cache.URL = value
	}
	if value := os.Getenv("IROHA_CACHE_BACKEND"); value != "" {
		cfg.Cache.Backend = value
	}
	if value := os.Getenv("IROHA_ALLOWED_ORIGINS"); value != "" {
		origins := make([]string, 0)
		for _, origin := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
		cfg.Server.AllowedOrigins = origins
	}
}
