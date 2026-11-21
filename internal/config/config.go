package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	Logger   LoggerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Temporal TemporalConfig
	Auth     AuthConfig
	Email    EmailConfig
	Didit    DiditConfig
}

type AppConfig struct {
	Name string
	Env  string
	Port int
}

type LoggerConfig struct {
	Level  string
	Pretty bool
}

type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	AutoMigrate     bool
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode,
	)
}

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
}

type TemporalConfig struct {
	HostPort  string
	Namespace string
	TaskQueue string
}

type AuthConfig struct {
	// JWTSecret signs access tokens (HS256). Must be at least 32 bytes.
	JWTSecret       string
	JWTIssuer       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int
	OTPTTL          time.Duration
	OTPMaxAttempts  int
	OTPResendDelay  time.Duration
}

type EmailConfig struct {
	// Provider is "brevo", "postmark", or "log" (log only prints emails; for local development).
	Provider              string
	From                  string
	FromName              string
	BrevoAPIKey           string
	PostmarkServerToken   string
	PostmarkMessageStream string
}

type DiditConfig struct {
	APIKey        string
	WorkflowID    string
	WebhookSecret string
	BaseURL       string
}

// devJWTSecret is only used when APP_ENV=development and JWT_SECRET is unset.
const devJWTSecret = "dev-only-insecure-jwt-secret-change-me-0123456789"

func Load() *Config {
	loadDotEnv(".env")

	env := getEnv("APP_ENV", "development")

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" && env == "development" {
		jwtSecret = devJWTSecret
	}

	emailProvider := "postmark"
	if env == "development" {
		emailProvider = "log"
	}

	return &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "rental-api"),
			Env:  env,
			Port: getEnvAsInt("APP_PORT", 8080),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Pretty: getEnvAsBool("LOG_PRETTY", env == "development"),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnvAsInt("POSTGRES_PORT", 5432),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:          getEnv("POSTGRES_DB", "rental"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("POSTGRES_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("POSTGRES_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("POSTGRES_CONN_MAX_LIFETIME", 30*time.Minute),
			AutoMigrate:     getEnvAsBool("POSTGRES_AUTO_MIGRATE", false),
		},
		Redis: RedisConfig{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 0),
			DialTimeout:  getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
		},
		Temporal: TemporalConfig{
			HostPort:  getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			TaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "rental-task-queue"),
		},
		Auth: AuthConfig{
			JWTSecret:       jwtSecret,
			JWTIssuer:       getEnv("JWT_ISSUER", "rental-api"),
			AccessTokenTTL:  getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTokenTTL: getEnvAsDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
			BcryptCost:      getEnvAsInt("BCRYPT_COST", 12),
			OTPTTL:          getEnvAsDuration("OTP_TTL", 5*time.Minute),
			OTPMaxAttempts:  getEnvAsInt("OTP_MAX_ATTEMPTS", 5),
			OTPResendDelay:  getEnvAsDuration("OTP_RESEND_DELAY", 60*time.Second),
		},
		Email: EmailConfig{
			Provider:              getEnv("EMAIL_PROVIDER", emailProvider),
			From:                  getEnv("EMAIL_FROM", "no-reply@example.com"),
			FromName:              getEnv("EMAIL_FROM_NAME", "Rental API"),
			BrevoAPIKey:           getEnv("BREVO_API_KEY", ""),
			PostmarkServerToken:   getEnv("POSTMARK_SERVER_TOKEN", ""),
			PostmarkMessageStream: getEnv("POSTMARK_MESSAGE_STREAM", "outbound"),
		},
		Didit: DiditConfig{
			APIKey:        getEnv("DIDIT_API_KEY", ""),
			WorkflowID:    getEnv("DIDIT_WORKFLOW_ID", ""),
			WebhookSecret: getEnv("DIDIT_WEBHOOK_SECRET", ""),
			BaseURL:       getEnv("DIDIT_BASE_URL", "https://verification.didit.me"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	valStr := getEnv(key, "")
	if val, err := time.ParseDuration(valStr); err == nil {
		return val
	}
	return fallback
}

func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
