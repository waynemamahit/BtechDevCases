package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
)

type duplicateDirectory struct{}

func (duplicateDirectory) Register(context.Context, user.Registration) (user.Account, error) {
	return user.Account{}, fmt.Errorf("%w: Duplicate entry 'ada@example.com' for key 'users_email_unique'", user.ErrDuplicateEmail)
}

func (duplicateDirectory) Authenticate(context.Context, string, string, time.Time) (user.Account, error) {
	return user.Account{}, user.ErrInvalidCredentials
}

func (duplicateDirectory) Activity(context.Context, string) (time.Time, bool, error) {
	return time.Time{}, false, nil
}

func (duplicateDirectory) Touch(context.Context, string, time.Time) error {
	return nil
}

func TestRegisterDuplicateHidesMySQLDetail(t *testing.T) {
	clock := token.SystemClock{}
	handler := NewServer(duplicateDirectory{}, nil, token.NewIssuer("secret", clock), clock, "http://localhost:3000").Handler()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"ada@example.com","password":"s3cret","confirmPassword":"s3cret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != user.ErrDuplicateEmail.Error() {
		t.Fatalf("error = %q", body.Error)
	}
	if strings.Contains(rec.Body.String(), "users_email_unique") || strings.Contains(rec.Body.String(), "Duplicate entry") {
		t.Fatalf("body leaked driver text: %s", rec.Body.String())
	}
}
