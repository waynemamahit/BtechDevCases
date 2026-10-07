package token_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"btechdevcases/internal/token"
)

func TestSignParseAndReject(t *testing.T) {
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	clock := token.NewManualClock(start)
	issuer := token.NewIssuer("test-secret", clock)

	signed, err := issuer.Sign("user-1", "ada@example.com")
	if err != nil {
		t.Fatal(err)
	}

	unverified, _, err := jwt.NewParser().ParseUnverified(signed, &token.Claims{})
	if err != nil {
		t.Fatal(err)
	}
	if unverified.Method == nil || unverified.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		t.Fatalf("alg = %v", unverified.Header["alg"])
	}

	claims, err := issuer.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("sub = %q", claims.Subject)
	}
	if claims.Email != "ada@example.com" {
		t.Fatalf("email = %q", claims.Email)
	}
	if claims.IssuedAt == nil || !claims.IssuedAt.Time.Equal(start) {
		t.Fatalf("iat = %v", claims.IssuedAt)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(start.Add(24*time.Hour)) {
		t.Fatalf("exp = %v", claims.ExpiresAt)
	}

	clock.Set(start.Add(15 * time.Minute))
	if _, err := issuer.Parse(signed); err != nil {
		t.Fatal(err)
	}

	clock.Set(start.Add(24*time.Hour - time.Second))
	if _, err := issuer.Parse(signed); err != nil {
		t.Fatal(err)
	}

	clock.Set(start.Add(24 * time.Hour))
	if _, err := issuer.Parse(signed); err == nil {
		t.Fatal("accepted an expired token")
	}

	clock.Set(start)
	if _, err := token.NewIssuer("other-secret", clock).Parse(signed); err == nil {
		t.Fatal("accepted a bad signature")
	}
}
