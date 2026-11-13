package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from the environment.
type Config struct {
	DBHost string
	DBPort int
	DBUser string
	DBPass string
	DBName string

	// Port is the HTTP port the Gin server listens on.
	Port int
}

// Load reads configuration from environment variables, applying defaults
// that match the docker-compose setup so the app runs out of the box.
func Load() (Config, error) {
	cfg := Config{
		DBHost: envOr("DB_HOST", "db"),
		DBUser: envOr("DB_USER", "simplehelp"),
		DBPass: envOr("DB_PASS", "simplehelp"),
		DBName: envOr("DB_NAME", "simplehelp"),
		Port:   envIntOr("PORT", 8080),
	}

	dbPort, err := envInt("DB_PORT", 5432)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	cfg.DBPort = dbPort

	return cfg, nil
}

// DSN returns a Postgres connection string for pgx.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName,
	)
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	v, err := envInt(key, def)
	if err != nil {
		return def
	}
	return v
}

func envInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}
