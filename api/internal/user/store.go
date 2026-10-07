package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"btechdevcases/internal/db"
)

const OpeningBalanceMinor int64 = 100000

var (
	ErrDuplicateEmail     = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Account struct {
	ID    string
	Email string
}

type Directory interface {
	Register(ctx context.Context, in Registration) (Account, error)
	Authenticate(ctx context.Context, email, password string, at time.Time) (Account, error)
	Activity(ctx context.Context, userID string) (time.Time, bool, error)
	Touch(ctx context.Context, userID string, at time.Time) error
}

type Store struct {
	db *sql.DB
}

func NewStore(sqlDB *sql.DB) *Store {
	return &Store{db: sqlDB}
}

func (s *Store) Register(ctx context.Context, in Registration) (Account, error) {
	if err := Validate(in); err != nil {
		return Account{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
	if err != nil {
		return Account{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback()

	id := uuid.NewString()
	queries := db.New(tx)
	err = queries.CreateUser(ctx, db.CreateUserParams{
		ID:           id,
		Email:        *in.Email,
		PasswordHash: string(hash),
	})
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return Account{}, fmt.Errorf("%w: %s", ErrDuplicateEmail, mysqlErr.Message)
		}
		return Account{}, err
	}
	if err := queries.CreateWallet(ctx, db.CreateWalletParams{
		UserID:       id,
		BalanceMinor: OpeningBalanceMinor,
	}); err != nil {
		return Account{}, err
	}
	if err := tx.Commit(); err != nil {
		return Account{}, err
	}
	return Account{ID: id, Email: *in.Email}, nil
}

func (s *Store) Authenticate(ctx context.Context, email, password string, at time.Time) (Account, error) {
	queries := db.New(s.db)
	row, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrInvalidCredentials
		}
		return Account{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)) != nil {
		return Account{}, ErrInvalidCredentials
	}
	if err := queries.SetLastActivity(ctx, db.SetLastActivityParams{
		LastActivityAt: sql.NullTime{Time: at, Valid: true},
		ID:             row.ID,
	}); err != nil {
		return Account{}, err
	}
	return Account{ID: row.ID, Email: row.Email}, nil
}

func (s *Store) Activity(ctx context.Context, userID string) (time.Time, bool, error) {
	at, err := db.New(s.db).GetLastActivity(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	if !at.Valid {
		return time.Time{}, false, nil
	}
	return at.Time, true, nil
}

func (s *Store) Touch(ctx context.Context, userID string, at time.Time) error {
	return db.New(s.db).SetLastActivity(ctx, db.SetLastActivityParams{
		LastActivityAt: sql.NullTime{Time: at, Valid: true},
		ID:             userID,
	})
}
