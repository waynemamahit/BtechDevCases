package user_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"golang.org/x/crypto/bcrypt"

	"btechdevcases/internal/db"
	"btechdevcases/internal/user"
)

func TestDuplicateEmail(t *testing.T) {
	sqlDB := startMySQL(t)
	ctx := context.Background()
	queries := db.New(sqlDB)
	store := user.NewStore(sqlDB)

	email := "Ada@example.com"
	password := "s3cret"
	first, err := store.Register(ctx, user.Registration{
		Email:           &email,
		Password:        &password,
		ConfirmPassword: &password,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(first.ID); err != nil {
		t.Fatal(err)
	}
	if first.Email != email {
		t.Fatalf("stored email %q", first.Email)
	}
	if storedBalance(t, sqlDB, first.ID) != user.OpeningBalanceMinor {
		t.Fatalf("opening balance %d", storedBalance(t, sqlDB, first.ID))
	}
	assertCounts(t, sqlDB, 1)

	stored, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PasswordHash == password {
		t.Fatal("stored secret is the submitted password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)); err != nil {
		t.Fatal(err)
	}

	replacement := "replacement"
	_, err = store.Register(ctx, user.Registration{
		Email:           &email,
		Password:        &replacement,
		ConfirmPassword: &replacement,
	})
	if !errors.Is(err, user.ErrDuplicateEmail) {
		t.Fatalf("duplicate insert error = %v", err)
	}
	if !strings.Contains(err.Error(), "users_email_unique") {
		t.Fatalf("duplicate error %q does not name users_email_unique", err)
	}
	assertCounts(t, sqlDB, 1)

	again, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID || again.PasswordHash != stored.PasswordHash {
		t.Fatal("duplicate insert replaced the original account")
	}
	if bcrypt.CompareHashAndPassword([]byte(again.PasswordHash), []byte(replacement)) == nil {
		t.Fatal("duplicate registration replaced the original password")
	}

	lower := "ada@example.com"
	second, err := store.Register(ctx, user.Registration{
		Email:           &lower,
		Password:        &password,
		ConfirmPassword: &password,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID || second.Email != lower {
		t.Fatalf("different-case email was folded into %+v", second)
	}
	if _, err := queries.GetUserByEmail(ctx, lower); err != nil {
		t.Fatal(err)
	}
	if storedBalance(t, sqlDB, second.ID) != user.OpeningBalanceMinor {
		t.Fatal("second account is missing its opening balance")
	}
	assertCounts(t, sqlDB, 2)
}

func startMySQL(t *testing.T) *sql.DB {
	t.Helper()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	ctx := context.Background()
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "db", "schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	container, err := mysql.Run(ctx, "mysql:8",
		mysql.WithDatabase("auth"),
		mysql.WithUsername("auth"),
		mysql.WithPassword("auth-password"),
		mysql.WithScripts(schemaPath),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate mysql: %v", err)
		}
	})
	dsn, err := container.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4", "loc=UTC")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	return sqlDB
}

func storedBalance(t *testing.T, sqlDB *sql.DB, userID string) int64 {
	t.Helper()
	var balance int64
	if err := sqlDB.QueryRow(`SELECT balance_minor FROM wallets WHERE user_id = ?`, userID).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	return balance
}

func assertCounts(t *testing.T, sqlDB *sql.DB, want int) {
	t.Helper()
	var users, wallets int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM wallets`).Scan(&wallets); err != nil {
		t.Fatal(err)
	}
	if users != want || wallets != want {
		t.Fatalf("users %d wallets %d, want %d", users, wallets, want)
	}
}
