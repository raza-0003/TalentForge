// Package config loads application configuration from defaults, an optional
// config.yaml file, and environment variables (which take precedence).
//
// Environment variables use the ATS_ prefix with underscores replacing dots,
// e.g. postgres.password -> ATS_POSTGRES_PASSWORD.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the fully-resolved application configuration.
type Config struct {
	Env      string
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
	Storage  StorageConfig
	Worker   WorkerConfig
	Log      LogConfig
}

// StorageConfig holds file-storage settings. Driver "local" writes under Dir
// (the API and worker must share it); driver "s3" uses the Bucket/Region and an
// optional Endpoint (for S3-compatible servers like MinIO).
type StorageConfig struct {
	Driver   string
	Dir      string
	Bucket   string
	Region   string
	Endpoint string
}

// JWTConfig holds token-signing settings.
type JWTConfig struct {
	Secret          string
	AccessTTLMin    int
	RefreshTTLHours int
}

// SMTPConfig holds outbound email settings. An empty Host puts the worker in
// dev mode where emails are logged instead of sent.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// WorkerConfig holds background-worker settings.
type WorkerConfig struct {
	Concurrency   int
	SweepCron     string // Asynq cron spec for the pending-notification sweeper
	SweepAfterMin int    // only sweep pending rows older than this many minutes
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string
	Port int
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN builds a libpq-style connection string.
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.DBName, p.SSLMode)
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// Addr returns the host:port Redis address.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string
}

// Load reads configuration and returns the resolved Config.
func Load() (*Config, error) {
	v := viper.New()

	setDefaults(v)

	// Environment variables override file values and defaults.
	v.SetEnvPrefix("ats")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	bindEnvs(v)

	// Optionally read config.yaml from the working directory or ./config.
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	if err := v.ReadInConfig(); err != nil {
		// A missing config file is fine — defaults + env vars still apply.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("env", "dev")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)

	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.user", "ats")
	v.SetDefault("postgres.password", "ats")
	v.SetDefault("postgres.dbname", "ats")
	v.SetDefault("postgres.sslmode", "disable")

	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("jwt.secret", "dev-insecure-change-me")
	v.SetDefault("jwt.accessttlmin", 15)
	v.SetDefault("jwt.refreshttlhours", 168) // 7 days

	v.SetDefault("smtp.host", "") // empty => log emails instead of sending
	v.SetDefault("smtp.port", 1025)
	v.SetDefault("smtp.username", "")
	v.SetDefault("smtp.password", "")
	v.SetDefault("smtp.from", "no-reply@ats.local")

	v.SetDefault("storage.driver", "local")
	v.SetDefault("storage.dir", "./storage")
	v.SetDefault("storage.bucket", "")
	v.SetDefault("storage.region", "")
	v.SetDefault("storage.endpoint", "")

	v.SetDefault("worker.concurrency", 10)
	v.SetDefault("worker.sweepcron", "@every 5m")
	v.SetDefault("worker.sweepaftermin", 15)

	v.SetDefault("log.level", "info")
}

// bindEnvs explicitly binds every key so that env-only overrides are picked up
// reliably by Unmarshal (a known Viper quirk with nested keys + AutomaticEnv).
func bindEnvs(v *viper.Viper) {
	keys := []string{
		"env",
		"server.host", "server.port",
		"postgres.host", "postgres.port", "postgres.user",
		"postgres.password", "postgres.dbname", "postgres.sslmode",
		"redis.host", "redis.port", "redis.password", "redis.db",
		"jwt.secret", "jwt.accessttlmin", "jwt.refreshttlhours",
		"smtp.host", "smtp.port", "smtp.username", "smtp.password", "smtp.from",
		"storage.driver", "storage.dir", "storage.bucket", "storage.region", "storage.endpoint",
		"worker.concurrency", "worker.sweepcron", "worker.sweepaftermin",
		"log.level",
	}
	for _, k := range keys {
		_ = v.BindEnv(k)
	}
}
