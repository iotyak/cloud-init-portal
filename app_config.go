package main

import (
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	PublicBaseURL     string
	TrustProxyHeaders bool
	StateFile         string
	StatusRateLimit   int
	WriteRateLimit    int
}

func LoadAppConfig() AppConfig {
	cfg := AppConfig{
		PublicBaseURL: strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")),
		StateFile:     strings.TrimSpace(os.Getenv("STATE_FILE")),
		StatusRateLimit: envIntOrDefault(
			"STATUS_RATE_LIMIT_PER_SEC",
			6,
		),
		WriteRateLimit: envIntOrDefault(
			"WRITE_RATE_LIMIT_PER_SEC",
			3,
		),
	}
	cfg.TrustProxyHeaders = envBool("TRUST_PROXY_HEADERS")
	return cfg
}

func envBool(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func envIntOrDefault(name string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
