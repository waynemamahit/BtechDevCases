package httpapi

import "fmt"

type Config struct {
	JWTSecret   string
	WebOrigin   string
	APIPort     string
	DatabaseDSN string
}

func LoadConfig(getenv func(string) string) (Config, error) {
	secret := getenv("JWT_SECRET")
	origin := getenv("WEB_ORIGIN")
	port := getenv("API_PORT")
	dsn := getenv("DATABASE_DSN")
	if secret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if origin == "" {
		return Config{}, fmt.Errorf("WEB_ORIGIN is required")
	}
	if dsn == "" {
		return Config{}, fmt.Errorf("DATABASE_DSN is required")
	}
	if port == "" {
		port = "8080"
	}
	return Config{
		JWTSecret:   secret,
		WebOrigin:   origin,
		APIPort:     port,
		DatabaseDSN: dsn,
	}, nil
}
