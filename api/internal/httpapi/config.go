package httpapi

import "fmt"

type Config struct {
	JWTSecret   string
	APIPort     string
	DatabaseDSN string
}

func LoadConfig(getenv func(string) string) (Config, error) {
	secret := getenv("JWT_SECRET")
	port := getenv("API_PORT")
	dsn := getenv("DATABASE_DSN")
	if secret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if dsn == "" {
		return Config{}, fmt.Errorf("DATABASE_DSN is required")
	}
	if port == "" {
		port = "8080"
	}
	return Config{
		JWTSecret:   secret,
		APIPort:     port,
		DatabaseDSN: dsn,
	}, nil
}
