package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"btechdevcases/internal/token"
)

func TestServerTimeouts(t *testing.T) {
	clock := token.SystemClock{}
	srv := NewServer(nil, nil, token.NewIssuer("secret", clock), clock)
	hs := srv.HTTPServer(":8080")
	if hs.ReadHeaderTimeout != 5*time.Second || hs.ReadTimeout != 10*time.Second || hs.WriteTimeout != 10*time.Second || hs.IdleTimeout != 60*time.Second {
		t.Fatalf("header %s read %s write %s idle %s", hs.ReadHeaderTimeout, hs.ReadTimeout, hs.WriteTimeout, hs.IdleTimeout)
	}

	var remaining time.Duration
	var hasDeadline bool
	srv.withTimeout(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		hasDeadline = ok
		remaining = time.Until(deadline)
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !hasDeadline {
		t.Fatal("request context has no deadline")
	}
	if remaining < 7*time.Second || remaining > 8*time.Second {
		t.Fatalf("deadline remaining %s", remaining)
	}
}
