package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

var Version = "dev"

type Config struct {
	App      AppConfig
	DB       DBConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
	S3       S3Config
	Auth     AuthConfig
	Payments PaymentConfig
	YooKassa YooKassaConfig
}

type AuthConfig struct {
	RateLimitRPS          float64
	RateLimitBurst        int
	ContactRateLimitRPS   float64
	ContactRateLimitBurst int
	ResendCooldown        time.Duration
}

type S3Config struct {
	Endpoint      string
	Region        string
	KeyID         string
	KeySecret     string
	Bucket        string
	PublicBaseURL string // e.g. https://global.s3.cloud/avi-dev
}

type AppConfig struct {
	Env            string
	Port           string
	TrustedOrigins []string
	SwaggerHosts   []string
	AdminUser      string
	AdminPassword  string
	TrustedProxies []string
}

type DBConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type SMTPConfig struct {
	Host           string
	Port           int
	From           string
	ContactTo      string
	User           string
	Password       string
	FrontendDomain string
}

type PaymentConfig struct {
	Provider                  string
	Currency                  string
	PromoteListingAmountMinor int64
	ReturnURL                 string
	ReceiptVatCode            int
	ReceiptPaymentSubject     string
	ReceiptPaymentMode        string
}

type YooKassaConfig struct {
	ShopID    string
	SecretKey string
	Enabled   bool
}

func Load() *Config {
	appEnv := getEnv("APP_ENV", "dev")
	smtpFrom := getEnv("SMTP_FROM", "noreply@avi.app")
	contactTo := getEmailRecipient("CONTACT_FORM_TO", appEnv, getEnv("RESEND_TEST_TO", smtpFrom))

	return &Config{
		App: AppConfig{
			Env:            appEnv,
			Port:           getEnv("APP_PORT", ":8080"),
			TrustedOrigins: getEnvCSV("CORS_TRUSTED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
			SwaggerHosts:   getEnvCSV("SWAGGER_HOSTS", []string{"localhost:8080"}),
			AdminUser:      getEnv("HTTP_BASIC_ADMIN_USER", "admin"),
			AdminPassword:  getEnv("HTTP_BASIC_ADMIN_PASSWORD", ""),
			TrustedProxies: getEnvCSVOptional("TRUSTED_PROXIES"),
		},
		DB: DBConfig{
			DSN:             getEnv("DB_DSN", "postgres://avi:avi@db:5432/avi?sslmode=disable"),
			MaxConns:        int32(getEnvInt("DB_MAX_CONNS", 10)),
			MinConns:        int32(getEnvInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: 5 * time.Minute,
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "dev-access-secret"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret"),
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
		},
		SMTP: SMTPConfig{
			Host:           getEnv("SMTP_HOST", "mailpit"),
			Port:           getEnvInt("SMTP_PORT", 1025),
			From:           smtpFrom,
			ContactTo:      contactTo,
			User:           getEnv("SMTP_USER", ""),
			Password:       getEnv("SMTP_PASSWORD", ""),
			FrontendDomain: getEnv("NEXT_PUBLIC_APP_URL", getEnv("FRONTEND_DOMAIN", "http://localhost:3000")),
		},
		S3: S3Config{
			Endpoint:      getEnv("S3_ENDPOINT", "https://s3.cloud"),
			Region:        getEnv("S3_REGION", "ru-central-1"),
			KeyID:         getEnv("S3_KEY_ID", ""),
			KeySecret:     getEnv("S3_KEY_SECRET", ""),
			Bucket:        getEnv("S3_BUCKET", ""),
			PublicBaseURL: getEnv("S3_PUBLIC_BASE_URL", ""),
		},
		Auth: AuthConfig{
			RateLimitRPS:          getEnvFloat64("AUTH_RATE_LIMIT_RPS", 0.5),
			RateLimitBurst:        getEnvInt("AUTH_RATE_LIMIT_BURST", 5),
			ContactRateLimitRPS:   getEnvFloat64("CONTACT_RATE_LIMIT_RPS", 5.0/60.0),
			ContactRateLimitBurst: getEnvInt("CONTACT_RATE_LIMIT_BURST", 5),
			ResendCooldown:        time.Duration(getEnvInt("AUTH_RESEND_COOLDOWN_SECONDS", 60)) * time.Second,
		},
		Payments: PaymentConfig{
			Provider:                  getEnv("PAYMENTS_PROVIDER", "yookassa"),
			Currency:                  getEnv("PAYMENT_CURRENCY", "RUB"),
			PromoteListingAmountMinor: int64(getEnvInt("PAYMENT_PROMOTE_LISTING_AMOUNT_MINOR", 10000)),
			ReturnURL:                 getEnv("YOOKASSA_RETURN_URL", "https://example.com/listing-promoted"),
			ReceiptVatCode:            getEnvInt("PAYMENT_RECEIPT_VAT_CODE", 1),
			ReceiptPaymentSubject:     getEnv("PAYMENT_RECEIPT_PAYMENT_SUBJECT", "service"),
			ReceiptPaymentMode:        getEnv("PAYMENT_RECEIPT_PAYMENT_MODE", "full_payment"),
		},
		YooKassa: YooKassaConfig{
			ShopID:    getEnv("YOOKASSA_SHOP_ID", ""),
			SecretKey: getEnv("YOOKASSA_SECRET_KEY", ""),
			Enabled:   getEnv("YOOKASSA_ENABLED", "false") == "true",
		},
	}
}

func getEmailRecipient(key, appEnv, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	if appEnv == "prod" {
		slog.Error("required email recipient env var is not set", "key", key)
		return ""
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvFloat64(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid duration env var, using fallback", "key", key, "value", v, "fallback", fallback, "error", err)
		return fallback
	}
	return d
}

func getEnvCSVOptional(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getEnvCSV(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		slog.Error("required env var is not set, using fallback", "key", key, "fallback", fallback)
		return fallback
	}

	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			out = append(out, s)
		}
	}

	if len(out) == 0 {
		slog.Error("required env var is empty, using fallback", "key", key, "fallback", fallback)
		return fallback
	}

	return out
}
