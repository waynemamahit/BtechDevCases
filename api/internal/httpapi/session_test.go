package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"btechdevcases/internal/token"
)

func TestMeRenewsAndExpiresWithTheClock(t *testing.T) {
	api := newAPI(t)
	id := registerAccount(t, api.handler, "ada@example.com", "s3cret")
	start := api.clock.Now()
	issued := loginToken(t, api.handler, "ada@example.com", "s3cret")

	api.clock.Set(start.Add(14 * time.Minute))
	renewed := getMe(api.handler, issued)
	if renewed.Code != http.StatusOK {
		t.Fatalf("renew status %d body %s", renewed.Code, renewed.Body.String())
	}
	var renewedBody struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}
	decodeJSON(t, renewed, &renewedBody)
	if renewedBody.ID != id || renewedBody.Email != "ada@example.com" || renewedBody.Token == "" {
		t.Fatalf("renew body %+v", renewedBody)
	}

	api.clock.Set(start.Add(28 * time.Minute))
	stillValid := getMe(api.handler, renewedBody.Token)
	if stillValid.Code != http.StatusOK {
		t.Fatalf("renewed token status %d body %s", stillValid.Code, stillValid.Body.String())
	}

	api.clock.Set(start.Add(29 * time.Minute))
	expired := getMe(api.handler, renewedBody.Token)
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("renewed token at 15 minutes status %d body %s", expired.Code, expired.Body.String())
	}
	assertNoToken(t, expired)
}

func TestMeRejectsUnrenewedCredentialAt15Minutes(t *testing.T) {
	api := newAPI(t)
	registerAccount(t, api.handler, "ada@example.com", "s3cret")
	start := api.clock.Now()
	issued := loginToken(t, api.handler, "ada@example.com", "s3cret")

	api.clock.Set(start.Add(15 * time.Minute))
	rec := getMe(api.handler, issued)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	assertNoToken(t, rec)
}

func TestMeRejectsMissingAndForeignTokens(t *testing.T) {
	api := newAPI(t)

	missing := getMe(api.handler, "")
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing status %d body %s", missing.Code, missing.Body.String())
	}
	assertNoToken(t, missing)

	foreign, err := token.NewIssuer("other-secret", api.clock).Sign("other-user", "ada@example.com")
	if err != nil {
		t.Fatal(err)
	}
	rejected := getMe(api.handler, foreign)
	if rejected.Code != http.StatusUnauthorized {
		t.Fatalf("foreign status %d body %s", rejected.Code, rejected.Body.String())
	}
	assertNoToken(t, rejected)
}

func TestCORSAllowsWebOrigin(t *testing.T) {
	api := newAPI(t)
	req := httptest.NewRequest(http.MethodOptions, "/login", nil)
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	rec := httptest.NewRecorder()
	api.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Fatalf("origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("methods = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Fatalf("headers = %q", got)
	}
}
