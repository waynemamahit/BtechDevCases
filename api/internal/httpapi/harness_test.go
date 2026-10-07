package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"btechdevcases/internal/httpapi"
	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
	"btechdevcases/internal/wallet"
)

const (
	testSecret = "test-secret"
	testOrigin = "http://localhost:5173"
)

type apiFixture struct {
	store   *memStore
	clock   *token.ManualClock
	issuer  *token.Issuer
	handler http.Handler
}

func newAPI(t *testing.T) apiFixture {
	t.Helper()
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	store := newMemStore()
	clock := token.NewManualClock(start)
	issuer := token.NewIssuer(testSecret, clock)
	ledger := wallet.New(store, clock)
	handler := httpapi.NewServer(store, ledger, issuer, clock, testOrigin).Handler()
	return apiFixture{store: store, clock: clock, issuer: issuer, handler: handler}
}

type memUser struct {
	id    string
	email string
	hash  string
}

type memTransfer struct {
	id          string
	senderID    string
	recipientID string
	amount      int64
	notes       string
	createdAt   time.Time
}

type memStore struct {
	mu        sync.Mutex
	byEmail   map[string]memUser
	byID      map[string]memUser
	balance   map[string]int64
	activity  map[string]time.Time
	transfers []memTransfer
}

func newMemStore() *memStore {
	return &memStore{
		byEmail:  map[string]memUser{},
		byID:     map[string]memUser{},
		balance:  map[string]int64{},
		activity: map[string]time.Time{},
	}
}

func (m *memStore) Register(ctx context.Context, in user.Registration) (user.Account, error) {
	if err := ctx.Err(); err != nil {
		return user.Account{}, err
	}
	if err := user.Validate(in); err != nil {
		return user.Account{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[*in.Email]; exists {
		return user.Account{}, user.ErrDuplicateEmail
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.MinCost)
	if err != nil {
		return user.Account{}, err
	}
	account := memUser{id: uuid.NewString(), email: *in.Email, hash: string(hash)}
	m.byEmail[account.email] = account
	m.byID[account.id] = account
	m.balance[account.id] = user.OpeningBalanceMinor
	return user.Account{ID: account.id, Email: account.email}, nil
}

func (m *memStore) Authenticate(ctx context.Context, email, password string, at time.Time) (user.Account, error) {
	if err := ctx.Err(); err != nil {
		return user.Account{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	account, ok := m.byEmail[email]
	if !ok || bcrypt.CompareHashAndPassword([]byte(account.hash), []byte(password)) != nil {
		return user.Account{}, user.ErrInvalidCredentials
	}
	m.activity[account.id] = at
	return user.Account{ID: account.id, Email: account.email}, nil
}

func (m *memStore) Activity(ctx context.Context, userID string) (time.Time, bool, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, false, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	at, ok := m.activity[userID]
	return at, ok, nil
}

func (m *memStore) Touch(ctx context.Context, userID string, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activity[userID] = at
	return nil
}

func (m *memStore) Balance(ctx context.Context, userID string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	bal, ok := m.balance[userID]
	if !ok {
		return 0, fmt.Errorf("wallet %s not found", userID)
	}
	return bal, nil
}

func (m *memStore) History(ctx context.Context, userID string) ([]wallet.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]memTransfer, 0)
	for _, tr := range m.transfers {
		if tr.senderID == userID || tr.recipientID == userID {
			rows = append(rows, tr)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].createdAt.Equal(rows[j].createdAt) {
			return rows[i].createdAt.Before(rows[j].createdAt)
		}
		return rows[i].id < rows[j].id
	})
	out := make([]wallet.Entry, 0, len(rows))
	for _, tr := range rows {
		entry := wallet.Entry{TransferID: tr.id, Amount: tr.amount, Notes: tr.notes}
		if tr.senderID == userID {
			entry.Direction = "sent"
			entry.Counterparty = m.byID[tr.recipientID].email
		} else {
			entry.Direction = "received"
			entry.Counterparty = m.byID[tr.senderID].email
		}
		out = append(out, entry)
	}
	return out, nil
}

func (m *memStore) InTx(ctx context.Context, fn func(wallet.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	balances := make(map[string]int64, len(m.balance))
	for id, bal := range m.balance {
		balances[id] = bal
	}
	tx := &memTx{
		store:     m,
		balances:  balances,
		transfers: append([]memTransfer(nil), m.transfers...),
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	m.balance = tx.balances
	m.transfers = tx.transfers
	return nil
}

type memTx struct {
	store     *memStore
	balances  map[string]int64
	transfers []memTransfer
}

func (t *memTx) FindUser(ctx context.Context, email string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	account, ok := t.store.byEmail[email]
	if !ok {
		return "", wallet.ErrNotFound
	}
	return account.id, nil
}

func (t *memTx) LockBalances(ctx context.Context, ids []string) (map[string]int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	out := make(map[string]int64, len(ordered))
	for _, id := range ordered {
		bal, ok := t.balances[id]
		if !ok {
			return nil, fmt.Errorf("wallet %s not found", id)
		}
		out[id] = bal
	}
	return out, nil
}

func (t *memTx) TransferByID(ctx context.Context, senderID, transferID string) (wallet.Record, bool, error) {
	if err := ctx.Err(); err != nil {
		return wallet.Record{}, false, err
	}
	for _, tr := range t.transfers {
		if tr.senderID == senderID && tr.id == transferID {
			return wallet.Record{
				TransferID:  tr.id,
				SenderID:    tr.senderID,
				RecipientID: tr.recipientID,
				Recipient:   t.store.byID[tr.recipientID].email,
				Amount:      tr.amount,
				Notes:       tr.notes,
				CreatedAt:   tr.createdAt,
			}, true, nil
		}
	}
	return wallet.Record{}, false, nil
}

func (t *memTx) SaveBalances(ctx context.Context, balances map[string]int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for id, bal := range balances {
		if _, ok := t.balances[id]; !ok {
			return fmt.Errorf("wallet %s not found", id)
		}
		t.balances[id] = bal
	}
	return nil
}

func (t *memTx) SaveTransfer(ctx context.Context, rec wallet.Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.transfers = append(t.transfers, memTransfer{
		id:          rec.TransferID,
		senderID:    rec.SenderID,
		recipientID: rec.RecipientID,
		amount:      rec.Amount,
		notes:       rec.Notes,
		createdAt:   rec.CreatedAt,
	})
	return nil
}

func (m *memStore) Lookup(email string) (memUser, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	account, ok := m.byEmail[email]
	return account, ok
}

func (m *memStore) PasswordMatches(email, password string) bool {
	m.mu.Lock()
	account, ok := m.byEmail[email]
	m.mu.Unlock()
	if !ok {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(account.hash), []byte(password)) == nil
}

func (m *memStore) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byEmail)
}

func postJSON(handler http.Handler, path, raw string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func getMe(handler http.Handler, rawToken string) *httptest.ResponseRecorder {
	return getAuth(handler, "/me", rawToken)
}

func getAuth(handler http.Handler, path, rawToken string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if rawToken != "" {
		req.Header.Set("Authorization", "Bearer "+rawToken)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func postAuth(handler http.Handler, path, rawToken, raw string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
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
