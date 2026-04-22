package config

import (
	"fmt"
	"os"
	"time"

	"github.com/go-core-fx/config"
)

type http struct {
	Address     string   `koanf:"address"`
	ProxyHeader string   `koanf:"proxy_header"`
	Proxies     []string `koanf:"proxies"`

	OpenAPI openAPIConfig `koanf:"openapi"`
}

type openAPIConfig struct {
	Enabled    bool   `koanf:"enabled"`
	PublicHost string `koanf:"public_host"`
	PublicPath string `koanf:"public_path"`
}

type databaseConfig struct {
	URL             string        `koanf:"url"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	MaxOpenConns    int           `koanf:"max_open_conns"`
	MaxIdleConns    int           `koanf:"max_idle_conns"`
}

type jwtConfig struct {
	Secret    string        `koanf:"secret"`
	AccessTTL time.Duration `koanf:"access_ttl"`
	Issuer    string        `koanf:"issuer"`
}

type storageConfig struct {
	URL      string        `koanf:"url"`
	LinksTTL time.Duration `koanf:"links_ttl"`
}

type attachmentsConfig struct {
	MaxSize uint64 `koanf:"max_size"`
}

type Config struct {
	HTTP        http              `koanf:"http"`
	Database    databaseConfig    `koanf:"database"`
	JWT         jwtConfig         `koanf:"jwt"`
	Storage     storageConfig     `koanf:"storage"`
	Attachments attachmentsConfig `koanf:"attachments"`
}

func Default() Config {
	//nolint:gosec,mnd // default values
	return Config{
		HTTP: http{
			Address:     "127.0.0.1:3000",
			ProxyHeader: "X-Forwarded-For",
			Proxies:     []string{},
			OpenAPI: openAPIConfig{
				Enabled:    true,
				PublicHost: "",
				PublicPath: "",
			},
		},
		Database: databaseConfig{
			URL:             "mariadb://bit-issues:bit-issues@127.0.0.1:3306/bit-issues?charset=utf8mb4&parseTime=True&loc=UTC&clientFoundRows=true",
			ConnMaxIdleTime: 0,
			ConnMaxLifetime: 0,
			MaxOpenConns:    0,
			MaxIdleConns:    0,
		},
		JWT: jwtConfig{
			Secret:    "secret",
			AccessTTL: time.Minute * 15,
			Issuer:    "bitissues.dev",
		},
		Storage: storageConfig{
			URL:      "s3://storage.bitissues.dev/attachments?endpoint=storage.bitissues.dev&region=us-east-1",
			LinksTTL: time.Minute * 15,
		},
		Attachments: attachmentsConfig{
			MaxSize: 10 * 1024 * 1024,
		},
	}
}

func New() (Config, error) {
	cfg := Default()

	options := []config.Option{}
	if yamlPath := os.Getenv("CONFIG_PATH"); yamlPath != "" {
		options = append(options, config.WithLocalYAML(yamlPath))
	}

	if err := config.Load(&cfg, options...); err != nil {
		return Config{}, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}
