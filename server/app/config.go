package app

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	HTTP     HTTPConfig              `koanf:"http"`
	Database DatabaseConfig          `koanf:"database"`
	TVDB     TVDBConfig              `koanf:"tvdb"`
	Auth     AuthConfig              `koanf:"auth"`
	Plugins  map[string]PluginConfig `koanf:"plugins"`
	Log      LogConfig               `koanf:"logging"`
}

type HTTPConfig struct {
	Addr string `koanf:"addr"`
}

type DatabaseConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	User     string `koanf:"user"`
	Password string `koanf:"password"`
	Database string `koanf:"database"`
	SSLMode  string `koanf:"ssl_mode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Database, d.SSLMode,
	)
}

type TVDBConfig struct {
	APIKey string `koanf:"apikey"`
	PIN    string `koanf:"pin"`
}

type PluginConfig struct {
	Path         string            `koanf:"path"`
	AllowedHosts []string          `koanf:"allowedhosts"`
	Config       map[string]string `koanf:"config"`
}

type AuthConfig struct {
	AdminToken string `koanf:"admintoken"`
}

type LogConfig struct {
	Level string `koanf:"level"`
}

func Load(cfgFile string) (*Config, error) {
	k := koanf.New(".")

	if cfgFile != "" {
		if err := k.Load(file.Provider(cfgFile), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load yaml config: %w", err)
		}
	}

	err := k.Load(env.Provider(".", env.Opt{
		Prefix: "APP_",
		TransformFunc: func(k, v string) (string, any) {
			k = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(k, "APP_")), "_", ".")

			if strings.Contains(v, " ") {
				return k, strings.Split(v, " ")
			}

			return k, v
		},
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("load env config: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}
	if len(c.Auth.AdminToken) < 16 {
		return fmt.Errorf("auth.admintoken of at least 16 characters is required (APP_AUTH_ADMINTOKEN)")
	}
	if c.HTTP.Addr == "" {
		c.HTTP.Addr = ":3000"
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	return nil
}
