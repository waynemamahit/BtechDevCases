package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

func TestLoginReturnsStableSubject(t *testing.T) {
	api := newAPI(t)
	id := registerAccount(t, api.handler, "ada@example.com", "s3cret")

	first := loginToken(t, api.handler, "ada@example.com", "s3cret")
	second := loginToken(t, api.handler, "ada@example.com", "s3cret")

	firstClaims, err := api.issuer.Parse(first)
	if err != nil {
		t.Fatal(err)
	}
	secondClaims, err := api.issuer.Parse(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstClaims.Subject != id || secondClaims.Subject != id {
		t.Fatalf("sub = %q and %q, want %q", firstClaims.Subject, secondClaims.Subject, id)
	}
	if firstClaims.Email != "ada@example.com" || secondClaims.Email != "ada@example.com" {
		t.Fatalf("email = %q and %q", firstClaims.Email, secondClaims.Email)
	}

	rec := postJSON(api.handler, "/login", `{"email":"ada@example.com","password":"s3cret"}`)
	var body map[string]any
	decodeJSON(t, rec, &body)
	if len(body) != 1 {
		t.Fatalf("login body keys = %v", body)
	}
	if _, ok := body["token"].(string); !ok {
		t.Fatal("login body is missing token")
	}
	assertActivity(t, api, id, api.clock.Now())
}

func TestLoginRejectsUnknownEmailAndWrongPassword(t *testing.T) {
	api := newAPI(t)
	id := registerAccount(t, api.handler, "ada@example.com", "s3cret")

	cases := []struct {
		name string
		body string
	}{
		{name: "unknown email", body: `{"email":"missing@example.com","password":"s3cret"}`},
		{name: "wrong password", body: `{"email":"ada@example.com","password":"nope"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postJSON(api.handler, "/login", tc.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			var body map[string]any
			decodeJSON(t, rec, &body)
			if body["error"] != "invalid credentials" {
				t.Fatalf("error = %v", body["error"])
			}
			assertNoToken(t, rec)
			if _, ok, err := api.store.Activity(context.Background(), id); err != nil || ok {
				t.Fatal("failed login recorded activity")
			}
		})
	}
}
