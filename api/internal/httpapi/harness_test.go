package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"btechdevcases/internal/httpapi"
	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
)

const (
	testSecret = "test-secret"
	testOrigin = "http://localhost:5173"
)

type apiFixture struct {
	store   *testDirectory
	clock   *token.ManualClock
	issuer  *token.Issuer
	handler http.Handler
}

func newAPI(t *testing.T) apiFixture {
	t.Helper()
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	store := newTestDirectory()
	clock := token.NewManualClock(start)
	issuer := token.NewIssuer(testSecret, clock)
	handler := httpapi.NewServer(store, issuer, testOrigin).Handler()
	return apiFixture{store: store, clock: clock, issuer: issuer, handler: handler}
}

type testAccount struct {
	id    string
	email string
	hash  string
}

type testDirectory struct {
	mu    sync.Mutex
	users map[string]testAccount
}

func newTestDirectory() *testDirectory {
	return &testDirectory{users: map[string]testAccount{}}
}

func (d *testDirectory) Register(_ context.Context, in user.Registration) (user.Account, error) {
	if err := user.Validate(in); err != nil {
		return user.Account{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.users[*in.Email]; exists {
		return user.Account{}, user.ErrDuplicateEmail
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
	if err != nil {
		return user.Account{}, err
	}
	account := testAccount{id: uuid.NewString(), email: *in.Email, hash: string(hash)}
	d.users[*in.Email] = account
	return user.Account{ID: account.id, Email: account.email}, nil
}

func (d *testDirectory) Authenticate(_ context.Context, email, password string) (user.Account, error) {
	d.mu.Lock()
	account, ok := d.users[email]
	d.mu.Unlock()
	if !ok || bcrypt.CompareHashAndPassword([]byte(account.hash), []byte(password)) != nil {
		return user.Account{}, user.ErrInvalidCredentials
	}
	return user.Account{ID: account.id, Email: account.email}, nil
}

func (d *testDirectory) Lookup(email string) (testAccount, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	account, ok := d.users[email]
	return account, ok
}

func (d *testDirectory) PasswordMatches(email, password string) bool {
	d.mu.Lock()
	account, ok := d.users[email]
	d.mu.Unlock()
	if !ok {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(account.hash), []byte(password)) == nil
}

func postJSON(handler http.Handler, path, raw string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func getMe(handler http.Handler, rawToken string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if rawToken != "" {
		req.Header.Set("Authorization", "Bearer "+rawToken)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
}

func assertNoToken(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]any
	decodeJSON(t, rec, &body)
	if _, ok := body["token"]; ok {
		t.Fatalf("response included a token: %s", rec.Body.String())
	}
}

func registerAccount(t *testing.T, handler http.Handler, email, password string) string {
	t.Helper()
	rec := postJSON(handler, "/register", `{"email":"`+email+`","password":"`+password+`","confirmPassword":"`+password+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	decodeJSON(t, rec, &body)
	if body.ID == "" || body.Email != email {
		t.Fatalf("register body %+v", body)
	}
	return body.ID
}

func loginToken(t *testing.T, handler http.Handler, email, password string) string {
	t.Helper()
	rec := postJSON(handler, "/login", `{"email":"`+email+`","password":"`+password+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Token string `json:"token"`
	}
	decodeJSON(t, rec, &body)
	if body.Token == "" {
		t.Fatal("login returned an empty token")
	}
	return body.Token
}
