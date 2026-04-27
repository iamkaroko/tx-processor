// internal/config/config.go
package config

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port           string         `env:"PORT" envDefault:"8080"`
	RedisConfig    RedisConfig    `envPrefix:"REDIS_"`
	DatabaseConfig DatabaseConfig `envPrefix:"DB_"`
}

type RedisConfig struct {
	RedisEnabled bool   `env:"ENABLED"  envDefault:"false"`
	RedisAddr    string `env:"ADDR"     envDefault:"localhost:6379"`
	RedisPw      string `env:"PASSWORD" envDefault:""`
	RedisDB      int    `env:"DB"       envDefault:"0"`
}

type DatabaseConfig struct {
	Host     string `env:"HOST"     envDefault:"localhost"`
	Port     string `env:"PORT"     envDefault:"5432"`
	User     string `env:"USER"     envDefault:"postgres"`
	Password string `env:"PASSWORD" envDefault:"postgres"`
	DBName   string `env:"NAME"     envDefault:"tx-processor"`
	SSLMode  string `env:"SSLMODE"  envDefault:"disable"`
}

func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

func New() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
