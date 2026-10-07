package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"btechdevcases/internal/token"
)

const welcomeAda = "Hello ada@example.com, welcome back"

func TestSession(t *testing.T) {
	t.Run("welcome", func(t *testing.T) {
		api := newAPI(t)
		registerAccount(t, api.handler, "ada@example.com", "s3cret")
		issued := loginToken(t, api.handler, "ada@example.com", "s3cret")
		rec := getMe(api.handler, issued)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		if rec.Body.String() != welcomeAda {
			t.Fatalf("body %q", rec.Body.String())
		}
	})

	t.Run("missing token", func(t *testing.T) {
		api := newAPI(t)
		rec := getMe(api.handler, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		if rec.Body.String() == welcomeAda {
			t.Fatal("missing token returned the welcome sentence")
		}
	})

	t.Run("activity inside the window", func(t *testing.T) {
		api := newAPI(t)
		id := registerAccount(t, api.handler, "ada@example.com", "s3cret")
		start := api.clock.Now()
		issued := loginToken(t, api.handler, "ada@example.com", "s3cret")

		api.clock.Set(start.Add(14 * time.Minute))
		first := getMe(api.handler, issued)
		if first.Code != http.StatusOK || first.Body.String() != welcomeAda {
			t.Fatalf("14m status %d body %q", first.Code, first.Body.String())
		}
		assertActivity(t, api, id, start.Add(14*time.Minute))

		api.clock.Set(start.Add(28 * time.Minute))
		second := getMe(api.handler, issued)
		if second.Code != http.StatusOK || second.Body.String() != welcomeAda {
			t.Fatalf("28m status %d body %q", second.Code, second.Body.String())
		}
		assertActivity(t, api, id, start.Add(28*time.Minute))

		api.clock.Set(start.Add(43 * time.Minute))
		idle := getMe(api.handler, issued)
		if idle.Code != http.StatusUnauthorized || idle.Body.String() == welcomeAda {
			t.Fatalf("43m status %d body %q", idle.Code, idle.Body.String())
		}
		assertActivity(t, api, id, start.Add(28*time.Minute))

		api.clock.Set(start.Add(44 * time.Minute))
		later := getMe(api.handler, issued)
		if later.Code != http.StatusUnauthorized || later.Body.String() == welcomeAda {
			t.Fatalf("44m status %d body %q", later.Code, later.Body.String())
		}
		assertActivity(t, api, id, start.Add(28*time.Minute))
	})

	t.Run("fifteen idle minutes", func(t *testing.T) {
		api := newAPI(t)
		id := registerAccount(t, api.handler, "ada@example.com", "s3cret")
		start := api.clock.Now()
		issued := loginToken(t, api.handler, "ada@example.com", "s3cret")

		api.clock.Set(start.Add(15 * time.Minute))
		rec := getMe(api.handler, issued)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() == welcomeAda {
			t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
		}
		assertActivity(t, api, id, start)

		api.clock.Set(start.Add(16 * time.Minute))
		again := getMe(api.handler, issued)
		if again.Code != http.StatusUnauthorized || again.Body.String() == welcomeAda {
			t.Fatalf("later status %d body %q", again.Code, again.Body.String())
		}
		assertActivity(t, api, id, start)
	})

	t.Run("canceled request", func(t *testing.T) {
		api := newAPI(t)
		id := registerAccount(t, api.handler, "ada@example.com", "s3cret")
		start := api.clock.Now()
		issued := loginToken(t, api.handler, "ada@example.com", "s3cret")

		api.clock.Set(start.Add(10 * time.Minute))
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer "+issued)
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		api.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusGatewayTimeout {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		if rec.Body.String() == welcomeAda {
			t.Fatal("timeout returned the welcome sentence")
		}
		assertActivity(t, api, id, start)

		api.clock.Set(start.Add(15 * time.Minute))
		later := getMe(api.handler, issued)
		if later.Code != http.StatusUnauthorized || later.Body.String() == welcomeAda {
			t.Fatalf("later status %d body %q", later.Code, later.Body.String())
		}
		assertActivity(t, api, id, start)
	})

	t.Run("foreign token", func(t *testing.T) {
		api := newAPI(t)
		foreign, err := token.NewIssuer("other-secret", api.clock).Sign("other-user", "ada@example.com")
		if err != nil {
			t.Fatal(err)
		}
		rec := getMe(api.handler, foreign)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() == welcomeAda {
			t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
		}
	})
}

func assertActivity(t *testing.T, api apiFixture, userID string, want time.Time) {
	t.Helper()
	at, ok, err := api.store.Activity(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !at.Equal(want) {
		t.Fatalf("activity = %v ok %v, want %v", at, ok, want)
	}
}
