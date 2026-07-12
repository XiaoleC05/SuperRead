package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	Port               string
	OxeliaGatewayMode  bool
	DefaultFetchMin    int
	GatewayHMACSecret  string
	FetchCronInterval string
	BriefingCronTime  string
	SMTPHost          string
	SMTPPort          int
	SMTPUser          string
	SMTPPassword      string
	FromEmail         string
	DefaultToEmail    string
}

var Cfg *Config

func Load() *Config {
	_ = godotenv.Load()

	Cfg = &Config{
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		Port:              getEnv("SUPERREAD_PORT", "8002"),
		OxeliaGatewayMode: getEnvBool("OXELIA_GATEWAY_MODE", false),
		DefaultFetchMin:   getEnvInt("DEFAULT_FETCH_INTERVAL_MIN", 30),
		GatewayHMACSecret:  getEnv("GATEWAY_HMAC_SECRET", ""),
		FetchCronInterval: getEnv("FETCH_CRON_INTERVAL", "@every 30m"),
		BriefingCronTime:  getEnv("BRIEFING_CRON_TIME", "0 8 * * *"),
		SMTPHost:          getEnv("SMTP_HOST", ""),
		SMTPPort:          getEnvInt("SMTP_PORT", 587),
		SMTPUser:          getEnv("SMTP_USER", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		FromEmail:         getEnv("FROM_EMAIL", ""),
		DefaultToEmail:    getEnv("DEFAULT_TO_EMAIL", ""),
	}

	if Cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if Cfg.GatewayHMACSecret == "" && Cfg.OxeliaGatewayMode {
		log.Println("[WARN] GATEWAY_HMAC_SECRET is empty in gateway mode - HMAC verification disabled")
	}

	return Cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
