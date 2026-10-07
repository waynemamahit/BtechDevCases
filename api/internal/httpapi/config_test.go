package httpapi_test

import (
	"strings"
	"testing"

	"btechdevcases/internal/httpapi"
)

func TestLoadConfigRejectsEmptySecretOriginOrDSN(t *testing.T) {
	_, err := httpapi.LoadConfig(func(key string) string {
		switch key {
		case "JWT_SECRET":
			return ""
		case "WEB_ORIGIN":
			return "*"
		case "DATABASE_DSN":
			return "user:pass@tcp(mysql:3306)/auth"
		default:
			return ""
		}
	})
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("empty JWT_SECRET error = %v", err)
	}

	_, err = httpapi.LoadConfig(func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "secret"
		case "WEB_ORIGIN":
			return ""
		case "DATABASE_DSN":
			return "user:pass@tcp(mysql:3306)/auth"
		default:
			return ""
		}
	})
	if err == nil || !strings.Contains(err.Error(), "WEB_ORIGIN") {
		t.Fatalf("empty WEB_ORIGIN error = %v", err)
	}

	_, err = httpapi.LoadConfig(func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "secret"
		case "WEB_ORIGIN":
			return "*"
		case "DATABASE_DSN":
			return ""
		default:
			return ""
		}
	})
	if err == nil || !strings.Contains(err.Error(), "DATABASE_DSN") {
		t.Fatalf("empty DATABASE_DSN error = %v", err)
	}
}

func TestLoadConfigDefaultsPort(t *testing.T) {
	cfg, err := httpapi.LoadConfig(func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "secret"
		case "WEB_ORIGIN":
			return "*"
		case "DATABASE_DSN":
			return "user:pass@tcp(mysql:3306)/auth"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIPort != "8080" {
		t.Fatalf("API_PORT = %q", cfg.APIPort)
	}
	if cfg.DatabaseDSN != "user:pass@tcp(mysql:3306)/auth" {
		t.Fatalf("DATABASE_DSN = %q", cfg.DatabaseDSN)
	}

	cfg, err = httpapi.LoadConfig(func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "secret"
		case "WEB_ORIGIN":
			return "*"
		case "API_PORT":
			return "9090"
		case "DATABASE_DSN":
			return "user:pass@tcp(mysql:3306)/auth"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIPort != "9090" {
		t.Fatalf("API_PORT = %q", cfg.APIPort)
	}
}
