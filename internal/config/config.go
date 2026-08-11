package config

import "os"

// Config holds minimal runtime configuration used by scaffold.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Env            string
	PostgresDSN    string
	RedisAddr      string
	HTTPPort       string
}

// Load returns a basic config (replace with env loading logic as needed).
func Load() *Config {
	c := &Config{
		ServiceName:    getEnv("SERVICE_NAME", "sample-api"),
		ServiceVersion: getEnv("SERVICE_VERSION", "0.1.0"),
		Env:            getEnv("ENVIRONMENT", "dev"),
		PostgresDSN:    getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		HTTPPort:       getEnv("HTTP_PORT", "8080"),
	}
	return c
}

func getEnv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}
