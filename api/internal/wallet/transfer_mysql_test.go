package wallet

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mysql"

	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
)

func TestTransferMySQL(t *testing.T) {
	sqlDB := startMySQL(t)
	ctx := context.Background()
	accounts := user.NewStore(sqlDB)

	adaEmail := "ada@example.com"
	bobEmail := "bob@example.com"
	password := "s3cret"
	ada, err := accounts.Register(ctx, user.Registration{
		Email:           &adaEmail,
		Password:        &password,
		ConfirmPassword: &password,
	})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := accounts.Register(ctx, user.Registration{
		Email:           &bobEmail,
		Password:        &password,
		ConfirmPassword: &password,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := storedBalance(t, sqlDB, ada.ID); got != 100000 {
		t.Fatalf("ada opening balance %d", got)
	}
	if got := storedBalance(t, sqlDB, bob.ID); got != 100000 {
		t.Fatalf("bob opening balance %d", got)
	}

	ledger := NewMySQL(sqlDB, token.SystemClock{})
	request := Request{
		TransferID: "t-mysql-1",
		Recipient:  bob.Email,
		Amount:     2500,
		Notes:      "lunch",
	}
	first, err := ledger.Transfer(ctx, ada.ID, ada.Email, request)
	if err != nil {
		t.Fatal(err)
	}
	if storedBalance(t, sqlDB, ada.ID) != 97500 || storedBalance(t, sqlDB, bob.ID) != 102500 {
		t.Fatalf("balances after transfer ada %d bob %d", storedBalance(t, sqlDB, ada.ID), storedBalance(t, sqlDB, bob.ID))
	}
	if got := countTransfers(t, sqlDB, ada.ID); got != 1 {
		t.Fatalf("transfers %d", got)
	}

	second, err := ledger.Transfer(ctx, ada.ID, ada.Email, request)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("replay %+v original %+v", second, first)
	}
	if storedBalance(t, sqlDB, ada.ID) != 97500 || storedBalance(t, sqlDB, bob.ID) != 102500 {
		t.Fatalf("balances after replay ada %d bob %d", storedBalance(t, sqlDB, ada.ID), storedBalance(t, sqlDB, bob.ID))
	}
	if got := countTransfers(t, sqlDB, ada.ID); got != 1 {
		t.Fatalf("transfers after replay %d", got)
	}
}

func TestTransferMySQLOverlappingRetry(t *testing.T) {
	sqlDB := startMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	accounts := user.NewStore(sqlDB)

	adaEmail := "ada@example.com"
	bobEmail := "bob@example.com"
	password := "s3cret"
	ada, err := accounts.Register(ctx, user.Registration{
		Email:           &adaEmail,
		Password:        &password,
		ConfirmPassword: &password,
	})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := accounts.Register(ctx, user.Registration{
		Email:           &bobEmail,
		Password:        &password,
		ConfirmPassword: &password,
	})
	if err != nil {
		t.Fatal(err)
	}

	ledger := NewMySQL(sqlDB, token.SystemClock{})
	request := Request{
		TransferID: "t-overlap",
		Recipient:  bob.Email,
		Amount:     2500,
		Notes:      "lunch",
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	snapshotted := make(chan struct{})
	var saveOnce sync.Once
	var releaseOnce sync.Once
	var lockCalls atomic.Int32
	releaseFn := func() { releaseOnce.Do(func() { close(release) }) }
	beforeSaveTransfer = func() {
		saveOnce.Do(func() { close(entered) })
		select {
		case <-release:
		case <-ctx.Done():
		}
	}
	beforeLockBalances = func() {
		if lockCalls.Add(1) == 2 {
			close(snapshotted)
		}
	}
	var wg sync.WaitGroup
	defer func() {
		releaseFn()
		wg.Wait()
		beforeSaveTransfer = nil
		beforeLockBalances = nil
	}()

	var first, second Result
	var err1, err2 error
	wg.Add(1)
	go func() {
		defer wg.Done()
		first, err1 = ledger.Transfer(ctx, ada.ID, ada.Email, request)
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		second, err2 = ledger.Transfer(ctx, ada.ID, ada.Email, request)
	}()
	select {
	case <-snapshotted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	releaseFn()
	wg.Wait()

	if err1 != nil || err2 != nil {
		t.Fatalf("first %v second %v", err1, err2)
	}
	if second != first {
		t.Fatalf("replay %+v original %+v", second, first)
	}
	if storedBalance(t, sqlDB, ada.ID) != 97500 || storedBalance(t, sqlDB, bob.ID) != 102500 {
		t.Fatalf("balances ada %d bob %d", storedBalance(t, sqlDB, ada.ID), storedBalance(t, sqlDB, bob.ID))
	}
	if got := countTransfers(t, sqlDB, ada.ID); got != 1 {
		t.Fatalf("transfers %d", got)
	}
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

func countTransfers(t *testing.T, sqlDB *sql.DB, senderID string) int {
	t.Helper()
	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM transfers WHERE sender_id = ?`, senderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
