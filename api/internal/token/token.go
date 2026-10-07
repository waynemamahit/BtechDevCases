package token

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const idleWindow = 15 * time.Minute

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type ManualClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewManualClock(now time.Time) *ManualClock {
	return &ManualClock{now: now}
}

func (c *ManualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *ManualClock) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret []byte
	clock  Clock
}

func NewIssuer(secret string, clock Clock) *Issuer {
	if clock == nil {
		clock = SystemClock{}
	}
	return &Issuer{secret: []byte(secret), clock: clock}
}

func (i *Issuer) Sign(userID, email string) (string, error) {
	now := i.clock.Now().UTC().Truncate(time.Second)
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(idleWindow)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (i *Issuer) Parse(raw string) (Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
		}
		return i.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(i.clock.Now),
	)
	if err != nil {
		return Claims{}, err
	}
	if parsed == nil || !parsed.Valid || claims.Subject == "" || claims.Email == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return Claims{}, errors.New("invalid token")
	}
	if !i.clock.Now().Before(claims.ExpiresAt.Time) {
		return Claims{}, errors.New("token expired")
	}
	return *claims, nil
}
