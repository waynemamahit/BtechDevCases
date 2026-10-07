package token_test

import (
	"testing"
	"time"

	"btechdevcases/internal/token"
)

func TestSignClaimsAndRejectsAtExactExpiry(t *testing.T) {
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	clock := token.NewManualClock(start)
	issuer := token.NewIssuer("test-secret", clock)

	signed, err := issuer.Sign("user-1", "ada@example.com")
	if err != nil {
		t.Fatal(err)
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
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(start.Add(15*time.Minute)) {
		t.Fatalf("exp = %v", claims.ExpiresAt)
	}

	clock.Set(start.Add(14 * time.Minute))
	if _, err := issuer.Parse(signed); err != nil {
		t.Fatal(err)
	}

	clock.Set(start.Add(15 * time.Minute))
	if _, err := issuer.Parse(signed); err == nil {
		t.Fatal("token was accepted at exactly 15 minutes")
	}
}
