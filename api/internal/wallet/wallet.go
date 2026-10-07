package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"unicode/utf8"
)

const (
	maxNotes      = 200
	maxTransferID = 64
)

var (
	ErrNotFound           = errors.New("recipient not found")
	ErrSelf               = errors.New("cannot transfer to self")
	ErrInvalidAmount      = errors.New("amount must be a positive integer")
	ErrNotesTooLong       = errors.New("notes must be at most 200 characters")
	ErrTransferID         = errors.New("transferId must be 1 to 64 characters")
	ErrRecipient          = errors.New("recipient is required")
	ErrInsufficient       = errors.New("insufficient funds")
	ErrConflict           = errors.New("transfer id reused with a different payload")
	ErrDuplicateTransfer  = errors.New("transfer id already exists")
	errIdempotentRollback = errors.New("rollback idempotent replay")
)

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

type Request struct {
	TransferID string
	Recipient  string
	Amount     int64
	Notes      string
}

type Result struct {
	TransferID string `json:"transferId"`
	Recipient  string `json:"recipient"`
	Amount     int64  `json:"amount"`
	Notes      string `json:"notes"`
}

type Entry struct {
	TransferID   string `json:"transferId"`
	Direction    string `json:"direction"`
	Counterparty string `json:"counterparty"`
	Amount       int64  `json:"amount"`
	Notes        string `json:"notes"`
}

type Snapshot struct {
	Balance   int64   `json:"balance"`
	Transfers []Entry `json:"transfers"`
}

type Record struct {
	TransferID  string
	SenderID    string
	RecipientID string
	Recipient   string
	Amount      int64
	Notes       string
	CreatedAt   time.Time
}

type Store interface {
	Balance(ctx context.Context, userID string) (int64, error)
	History(ctx context.Context, userID string) ([]Entry, error)
	InTx(ctx context.Context, fn func(Tx) error) error
}

type Tx interface {
	FindUser(ctx context.Context, email string) (string, error)
	LockBalances(ctx context.Context, ids []string) (map[string]int64, error)
	TransferByID(ctx context.Context, senderID, transferID string) (Record, bool, error)
	SaveBalances(ctx context.Context, balances map[string]int64) error
	SaveTransfer(ctx context.Context, rec Record) error
}

type Service struct {
	store Store
	clock Clock
}

func New(store Store, clock Clock) *Service {
	if clock == nil {
		clock = systemClock{}
	}
	return &Service{store: store, clock: clock}
}

func NewMySQL(sqlDB *sql.DB, clock Clock) *Service {
	return New(newSQLStore(sqlDB), clock)
}

func (s *Service) Read(ctx context.Context, userID string) (Snapshot, error) {
	balance, err := s.store.Balance(ctx, userID)
	if err != nil {
		return Snapshot{}, err
	}
	entries, err := s.store.History(ctx, userID)
	if err != nil {
		return Snapshot{}, err
	}
	if entries == nil {
		entries = []Entry{}
	}
	return Snapshot{Balance: balance, Transfers: entries}, nil
}

func (s *Service) Transfer(ctx context.Context, senderID, senderEmail string, in Request) (Result, error) {
	if err := validate(in); err != nil {
		return Result{}, err
	}
	if in.Recipient == senderEmail {
		return Result{}, ErrSelf
	}

	var result Result
	err := s.store.InTx(ctx, func(tx Tx) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		recipientID, findErr := tx.FindUser(ctx, in.Recipient)
		unknown := errors.Is(findErr, ErrNotFound)
		if findErr != nil && !unknown {
			return findErr
		}

		var balances map[string]int64
		if !unknown {
			var err error
			balances, err = tx.LockBalances(ctx, []string{senderID, recipientID})
			if err != nil {
				return err
			}
		}

		existing, found, err := tx.TransferByID(ctx, senderID, in.TransferID)
		if err != nil {
			return err
		}
		if found {
			if !unknown && sameTransfer(existing, recipientID, in) {
				result = transferResult(existing)
				return nil
			}
			return ErrConflict
		}
		if unknown {
			return ErrNotFound
		}
		if in.Amount > balances[senderID] {
			return ErrInsufficient
		}
		balances[senderID] -= in.Amount
		balances[recipientID] += in.Amount
		if err := tx.SaveBalances(ctx, balances); err != nil {
			return err
		}
		rec := Record{
			TransferID:  in.TransferID,
			SenderID:    senderID,
			RecipientID: recipientID,
			Recipient:   in.Recipient,
			Amount:      in.Amount,
			Notes:       in.Notes,
			CreatedAt:   s.clock.Now().UTC(),
		}
		if err := tx.SaveTransfer(ctx, rec); err != nil {
			if !errors.Is(err, ErrDuplicateTransfer) {
				return err
			}
			return replayDuplicate(ctx, tx, senderID, recipientID, in, &result)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		result = Result{
			TransferID: rec.TransferID,
			Recipient:  rec.Recipient,
			Amount:     rec.Amount,
			Notes:      rec.Notes,
		}
		return nil
	})
	if errors.Is(err, errIdempotentRollback) {
		return result, nil
	}
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func replayDuplicate(ctx context.Context, tx Tx, senderID, recipientID string, in Request, result *Result) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	existing, found, err := tx.TransferByID(ctx, senderID, in.TransferID)
	if err != nil {
		return err
	}
	if !found {
		return ErrDuplicateTransfer
	}
	if !sameTransfer(existing, recipientID, in) {
		return ErrConflict
	}
	*result = transferResult(existing)
	return errIdempotentRollback
}

func sameTransfer(existing Record, recipientID string, in Request) bool {
	return existing.RecipientID == recipientID && existing.Amount == in.Amount && existing.Notes == in.Notes
}

func transferResult(existing Record) Result {
	return Result{
		TransferID: existing.TransferID,
		Recipient:  existing.Recipient,
		Amount:     existing.Amount,
		Notes:      existing.Notes,
	}
}

func validate(in Request) error {
	if in.TransferID == "" || utf8.RuneCountInString(in.TransferID) > maxTransferID {
		return ErrTransferID
	}
	if in.Recipient == "" {
		return ErrRecipient
	}
	if in.Amount <= 0 {
		return ErrInvalidAmount
	}
	if utf8.RuneCountInString(in.Notes) > maxNotes {
		return ErrNotesTooLong
	}
	return nil
}
