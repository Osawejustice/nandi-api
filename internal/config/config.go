package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config is the process-wide configuration loaded from .env and the environment.
type Config struct {
	App       AppConfig
	Server    ServerConfig
	Log       LogConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	CORS      CORSConfig
	AI        AIConfig
	AT        AfricaTalkingConfig
	Evolution EvolutionConfig
	Provider  ProviderConfig
	Webhook   WebhookConfig
}

type AppConfig struct {
	Env string
}

type ServerConfig struct {
	Host            string
	Port            string
	ShutdownTimeout time.Duration
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

type LogConfig struct {
	Level string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	URL      string
}

func (d DatabaseConfig) DSN() string {
	if strings.TrimSpace(d.URL) != "" {
		return d.URL
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		d.Host, d.User, d.Password, d.Name, d.Port, d.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type CORSConfig struct {
	Origins []string
}

type AIConfig struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
}

type AfricaTalkingConfig struct {
	Username string
	APIKey   string
	SenderID string
	Sandbox  bool
}

type EvolutionConfig struct {
	BaseURL  string
	APIKey   string
	Instance string
}

type ProviderConfig struct {
	PrimarySMS       string
	FailoverSMS      string
	PrimaryWhatsApp  string
	FailoverWhatsApp string
}

type WebhookConfig struct {
	Secret string
}

// Load reads optional .env files, then overlays environment variables via Viper.
func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	cfg := &Config{
		App: AppConfig{
			Env: v.GetString("APP_ENV"),
		},
		Server: ServerConfig{
			Host:            v.GetString("SERVER_HOST"),
			Port:            v.GetString("SERVER_PORT"),
			ShutdownTimeout: v.GetDuration("SERVER_SHUTDOWN_TIMEOUT"),
		},
		Log: LogConfig{
			Level: v.GetString("LOG_LEVEL"),
		},
		Database: DatabaseConfig{
			Host:     v.GetString("DB_HOST"),
			Port:     v.GetString("DB_PORT"),
			User:     v.GetString("DB_USER"),
			Password: v.GetString("DB_PASSWORD"),
			Name:     v.GetString("DB_NAME"),
			SSLMode:  v.GetString("DB_SSLMODE"),
			URL:      firstNonEmpty(v.GetString("DATABASE_URL"), v.GetString("DB_URL")),
		},
		Redis: RedisConfig{
			Addr:     v.GetString("REDIS_ADDR"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:     v.GetString("JWT_SECRET"),
			AccessTTL:  v.GetDuration("JWT_ACCESS_TTL"),
			RefreshTTL: v.GetDuration("JWT_REFRESH_TTL"),
		},
		CORS: CORSConfig{
			Origins: splitCSV(v.GetString("CORS_ALLOWED_ORIGINS")),
		},
		AI: AIConfig{
			Provider: v.GetString("AI_PROVIDER"),
			APIKey:   v.GetString("AI_API_KEY"),
			Model:    v.GetString("AI_MODEL"),
			BaseURL:  v.GetString("AI_BASE_URL"),
		},
		AT: AfricaTalkingConfig{
			Username: v.GetString("AT_USERNAME"),
			APIKey:   v.GetString("AT_API_KEY"),
			SenderID: v.GetString("AT_SENDER_ID"),
			Sandbox:  v.GetBool("AT_SANDBOX"),
		},
		Evolution: EvolutionConfig{
			BaseURL:  v.GetString("EVOLUTION_BASE_URL"),
			APIKey:   v.GetString("EVOLUTION_API_KEY"),
			Instance: v.GetString("EVOLUTION_INSTANCE"),
		},
		Provider: ProviderConfig{
			PrimarySMS:       v.GetString("PROVIDER_PRIMARY_SMS"),
			FailoverSMS:      v.GetString("PROVIDER_FAILOVER_SMS"),
			PrimaryWhatsApp:  v.GetString("PROVIDER_PRIMARY_WHATSAPP"),
			FailoverWhatsApp: v.GetString("PROVIDER_FAILOVER_WHATSAPP"),
		},
		Webhook: WebhookConfig{
			Secret: v.GetString("WEBHOOK_SECRET"),
		},
	}

	if cfg.Server.Port == "" {
		return nil, fmt.Errorf("SERVER_PORT is required")
	}
	if cfg.Server.ShutdownTimeout <= 0 {
		cfg.Server.ShutdownTimeout = 10 * time.Second
	}

	return cfg, nil
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.App.Env, "production")
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("SERVER_HOST", "0.0.0.0")
	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "10s")

	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_USER", "nandi")
	v.SetDefault("DB_PASSWORD", "nandi")
	v.SetDefault("DB_NAME", "nandi")
	v.SetDefault("DB_SSLMODE", "disable")

	v.SetDefault("REDIS_ADDR", "localhost:6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)

	v.SetDefault("JWT_SECRET", "change-me-in-production")
	v.SetDefault("JWT_ACCESS_TTL", "15m")
	v.SetDefault("JWT_REFRESH_TTL", "168h")

	v.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")

	v.SetDefault("AI_PROVIDER", "groq")
	v.SetDefault("AI_MODEL", "llama-3.1-8b-instant")
	v.SetDefault("AI_BASE_URL", "https://api.groq.com/openai/v1")

	v.SetDefault("AT_SANDBOX", true)

	v.SetDefault("PROVIDER_PRIMARY_SMS", "africastalking")
	v.SetDefault("PROVIDER_FAILOVER_SMS", "stub")
	v.SetDefault("PROVIDER_PRIMARY_WHATSAPP", "evolution")
	v.SetDefault("PROVIDER_FAILOVER_WHATSAPP", "stub")
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
