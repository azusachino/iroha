package config

import (
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

// defaultCacheURL points at the valkey instance from ops/local-dev/compose.yaml.
// Valkey speaks the Redis protocol, so a redis:// URL works unmodified.
const defaultCacheURL = "redis://localhost:6379/0"

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Storage  StorageConfig  `toml:"storage"`
	Auth     AuthConfig     `toml:"auth"`
	Cache    CacheConfig    `toml:"cache"`
}

type ServerConfig struct {
	Addr string `toml:"addr"`
}

type DatabaseConfig struct {
	URL string `toml:"url"`
}

type StorageConfig struct {
	DataDir string `toml:"data_dir"`
}

type AuthConfig struct {
	LocalNoAuth bool   `toml:"local_no_auth"`
	ImportToken string `toml:"import_token"`
}

type CacheConfig struct {
	URL string `toml:"url"`
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
	return cfg, nil
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Addr: "127.0.0.1:8080",
		},
		Database: DatabaseConfig{
			URL: "postgres://iroha:iroha_dev@localhost:5432/iroha?sslmode=disable",
		},
		Storage: StorageConfig{
			DataDir: ".iroha-data",
		},
		Auth: AuthConfig{
			LocalNoAuth: true,
		},
		Cache: CacheConfig{
			URL: defaultCacheURL,
		},
	}
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("IROHA_SERVER_ADDR"); value != "" {
		cfg.Server.Addr = value
	}
	if value := os.Getenv("IROHA_DATABASE_URL"); value != "" {
		cfg.Database.URL = value
	}
	if value := os.Getenv("IROHA_DATA_DIR"); value != "" {
		cfg.Storage.DataDir = value
	}
	if value := os.Getenv("IROHA_LOCAL_NO_AUTH"); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Auth.LocalNoAuth = parsed
		}
	}
	if value := os.Getenv("IROHA_IMPORT_TOKEN"); value != "" {
		cfg.Auth.ImportToken = value
	}
	if value := os.Getenv("IROHA_VALKEY_URL"); value != "" {
		cfg.Cache.URL = value
	}
}
