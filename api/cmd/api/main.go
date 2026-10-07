package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"btechdevcases/internal/db"
	"btechdevcases/internal/httpapi"
	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := httpapi.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}

	sqlDB, err := sql.Open("mysql", cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}

	store := user.NewStore(db.New(sqlDB))
	issuer := token.NewIssuer(cfg.JWTSecret, token.SystemClock{})
	server := httpapi.NewServer(store, issuer, cfg.WebOrigin)
	return http.ListenAndServe(":"+cfg.APIPort, server.Handler())
}
