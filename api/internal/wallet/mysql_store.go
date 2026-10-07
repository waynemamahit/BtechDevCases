package wallet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/go-sql-driver/mysql"

	"btechdevcases/internal/db"
)

// beforeLockBalances and beforeSaveTransfer let a test pause inside a transfer
// transaction. Production leaves both nil.
var (
	beforeLockBalances func()
	beforeSaveTransfer func()
)

type sqlStore struct {
	db *sql.DB
}

func newSQLStore(sqlDB *sql.DB) *sqlStore {
	return &sqlStore{db: sqlDB}
}

func (s *sqlStore) Balance(ctx context.Context, userID string) (int64, error) {
	row, err := db.New(s.db).GetWallet(ctx, userID)
	if err != nil {
		return 0, err
	}
	return row.BalanceMinor, nil
}

func (s *sqlStore) History(ctx context.Context, userID string) ([]Entry, error) {
	rows, err := db.New(s.db).ListTransfersForUser(ctx, db.ListTransfersForUserParams{UserID: userID})
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entry := Entry{
			TransferID: row.ID,
			Amount:     row.AmountMinor,
			Notes:      row.Notes,
		}
		if row.SenderID == userID {
			entry.Direction = "sent"
			entry.Counterparty = row.RecipientEmail
		} else {
			entry.Direction = "received"
			entry.Counterparty = row.SenderEmail
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *sqlStore) InTx(ctx context.Context, fn func(Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(&sqlTx{q: db.New(tx)}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := ctx.Err(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type sqlTx struct {
	q *db.Queries
}

func (t *sqlTx) FindUser(ctx context.Context, email string) (string, error) {
	row, err := t.q.GetUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func (t *sqlTx) LockBalances(ctx context.Context, ids []string) (map[string]int64, error) {
	if beforeLockBalances != nil {
		beforeLockBalances()
	}
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	out := make(map[string]int64, len(ordered))
	for _, id := range ordered {
		row, err := t.q.LockWallet(ctx, id)
		if err != nil {
			return nil, err
		}
		out[id] = row.BalanceMinor
	}
	return out, nil
}

func (t *sqlTx) TransferByID(ctx context.Context, senderID, transferID string) (Record, bool, error) {
	row, err := t.q.GetTransferBySenderAndID(ctx, db.GetTransferBySenderAndIDParams{
		SenderID: senderID,
		ID:       transferID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	email, err := t.q.GetEmailByID(ctx, row.RecipientID)
	if err != nil {
		return Record{}, false, err
	}
	return Record{
		TransferID:  row.ID,
		SenderID:    row.SenderID,
		RecipientID: row.RecipientID,
		Recipient:   email,
		Amount:      row.AmountMinor,
		Notes:       row.Notes,
		CreatedAt:   row.CreatedAt,
	}, true, nil
}

func (t *sqlTx) SaveBalances(ctx context.Context, balances map[string]int64) error {
	for userID, balance := range balances {
		if err := t.q.SetBalance(ctx, db.SetBalanceParams{
			BalanceMinor: balance,
			UserID:       userID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (t *sqlTx) SaveTransfer(ctx context.Context, rec Record) error {
	if beforeSaveTransfer != nil {
		beforeSaveTransfer()
	}
	err := t.q.InsertTransfer(ctx, db.InsertTransferParams{
		SenderID:    rec.SenderID,
		ID:          rec.TransferID,
		RecipientID: rec.RecipientID,
		AmountMinor: rec.Amount,
		Notes:       rec.Notes,
		CreatedAt:   rec.CreatedAt,
	})
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return fmt.Errorf("%w", ErrDuplicateTransfer)
	}
	return err
}
