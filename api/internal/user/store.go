package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"btechdevcases/internal/db"
)

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
	Authenticate(ctx context.Context, email, password string) (Account, error)
}

type Store struct {
	queries *db.Queries
}

func NewStore(queries *db.Queries) *Store {
	return &Store{queries: queries}
}

func (s *Store) Register(ctx context.Context, in Registration) (Account, error) {
	if err := Validate(in); err != nil {
		return Account{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
	if err != nil {
		return Account{}, err
	}

	id := uuid.NewString()
	err = s.queries.CreateUser(ctx, db.CreateUserParams{
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

	return Account{ID: id, Email: *in.Email}, nil
}

func (s *Store) Authenticate(ctx context.Context, email, password string) (Account, error) {
	row, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrInvalidCredentials
		}
		return Account{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)) != nil {
		return Account{}, ErrInvalidCredentials
	}
	return Account{ID: row.ID, Email: row.Email}, nil
}
