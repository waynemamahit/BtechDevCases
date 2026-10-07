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

func TestDuplicateEmailRejectedByMySQLUniqueKey(t *testing.T) {
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
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("terminate mysql: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4")
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

	queries := db.New(sqlDB)
	store := user.NewStore(queries)
	email := "Ada@Example.com"
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
}
