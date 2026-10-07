package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"btechdevcases/internal/httpapi"
	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
	"btechdevcases/internal/wallet"
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

	clock := token.SystemClock{}
	store := user.NewStore(sqlDB)
	ledger := wallet.NewMySQL(sqlDB, clock)
	issuer := token.NewIssuer(cfg.JWTSecret, clock)
	api := httpapi.NewServer(store, ledger, issuer, clock, cfg.WebOrigin)
	return api.HTTPServer(":" + cfg.APIPort).ListenAndServe()
}
