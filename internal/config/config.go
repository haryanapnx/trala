package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string        `env:"APP_PORT" envDefault:"8080"`
	AppEnv      string        `env:"APP_ENV" envDefault:"development"`
	LogLevel    string        `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL string        `env:"DATABASE_URL,required"`
	JWTSecret   string        `env:"JWT_SECRET,required"`
	JWTExpiry   time.Duration `env:"JWT_EXPIRY" envDefault:"24h"`

	// Database connection pool.
	DBMaxConns        int32         `env:"DB_MAX_CONNS" envDefault:"10"`
	DBMinConns        int32         `env:"DB_MIN_CONNS" envDefault:"1"`
	DBMaxConnLifetime time.Duration `env:"DB_MAX_CONN_LIFETIME" envDefault:"1h"`
	DBMaxConnIdleTime time.Duration `env:"DB_MAX_CONN_IDLE_TIME" envDefault:"30m"`
	// DBLockTimeout bounds how long a transaction waits for a row lock before
	// failing (mapped to 409 + Retry-After for in-flight duplicate requests).
	DBLockTimeout time.Duration `env:"DB_LOCK_TIMEOUT" envDefault:"2s"`
}

// Load reads configuration from the environment, loading a local .env file when
// present. Missing required values produce a descriptive error.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
