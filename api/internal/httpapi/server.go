package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
	"btechdevcases/internal/wallet"
)

const (
	idleLimit      = 15 * time.Minute
	requestTimeout = 8 * time.Second
)

type Server struct {
	users   user.Directory
	wallets *wallet.Service
	tokens  *token.Issuer
	clock   token.Clock
	origin  string
}

func NewServer(users user.Directory, wallets *wallet.Service, tokens *token.Issuer, clock token.Clock, origin string) *Server {
	if clock == nil {
		clock = token.SystemClock{}
	}
	return &Server{users: users, wallets: wallets, tokens: tokens, clock: clock, origin: origin}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", s.handleRegister)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /me", s.protected(s.serveMe))
	mux.HandleFunc("GET /wallet", s.protected(s.serveWallet))
	mux.HandleFunc("POST /transfers", s.protected(s.serveTransfer))
	return s.withTimeout(s.withCORS(mux))
}

func (s *Server) HTTPServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (s *Server) withTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type errorBody struct {
	Error string `json:"error"`
}

type accountResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var in user.Registration
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid JSON"})
		return
	}
	account, err := s.users.Register(r.Context(), in)
	if err != nil {
		if isTimeout(err) {
			writeTimeout(w)
			return
		}
		writeRegisterError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, accountResponse{ID: account.ID, Email: account.Email})
}

func writeRegisterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrDuplicateEmail):
		writeJSON(w, http.StatusConflict, errorBody{Error: user.ErrDuplicateEmail.Error()})
	case errors.Is(err, user.ErrMissingEmail),
		errors.Is(err, user.ErrMissingPassword),
		errors.Is(err, user.ErrMissingConfirmPassword),
		errors.Is(err, user.ErrInvalidEmail),
		errors.Is(err, user.ErrEmptyPassword),
		errors.Is(err, user.ErrPasswordMismatch):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not register"})
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid JSON"})
		return
	}
	account, err := s.users.Authenticate(r.Context(), in.Email, in.Password, s.clock.Now())
	if err != nil {
		if isTimeout(err) {
			writeTimeout(w)
			return
		}
		if errors.Is(err, user.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, errorBody{Error: "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not login"})
		return
	}
	signed, err := s.tokens.Sign(account.ID, account.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not sign token"})
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{Token: signed})
}

func (s *Server) protected(next func(http.ResponseWriter, *http.Request, user.Account)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Err() != nil {
			writeTimeout(w)
			return
		}
		raw, ok := bearerToken(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorBody{Error: "unauthorized"})
			return
		}
		claims, err := s.tokens.Parse(raw)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errorBody{Error: "unauthorized"})
			return
		}
		if r.Context().Err() != nil {
			writeTimeout(w)
			return
		}
		last, active, err := s.users.Activity(r.Context(), claims.Subject)
		if err != nil {
			if isTimeout(err) {
				writeTimeout(w)
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not load session"})
			return
		}
		if !active || !s.clock.Now().Before(last.Add(idleLimit)) {
			writeJSON(w, http.StatusUnauthorized, errorBody{Error: "unauthorized"})
			return
		}

		account := user.Account{ID: claims.Subject, Email: claims.Email}
		rec := &statusWriter{ResponseWriter: w}
		next(rec, r, account)
		if rec.status >= 200 && rec.status < 300 && r.Context().Err() == nil {
			_ = s.users.Touch(r.Context(), account.ID, s.clock.Now())
		}
	}
}

func (s *Server) serveMe(w http.ResponseWriter, r *http.Request, account user.Account) {
	if r.Context().Err() != nil {
		writeTimeout(w)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "Hello "+account.Email+", welcome back")
}

func (s *Server) serveWallet(w http.ResponseWriter, r *http.Request, account user.Account) {
	if r.Context().Err() != nil {
		writeTimeout(w)
		return
	}
	snap, err := s.wallets.Read(r.Context(), account.ID)
	if err != nil {
		if isTimeout(err) {
			writeTimeout(w)
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not read wallet"})
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

type transferBody struct {
	TransferID *string         `json:"transferId"`
	Recipient  *string         `json:"recipient"`
	Amount     json.RawMessage `json:"amount"`
	Notes      *string         `json:"notes"`
}

func (s *Server) serveTransfer(w http.ResponseWriter, r *http.Request, account user.Account) {
	if r.Context().Err() != nil {
		writeTimeout(w)
		return
	}
	var body transferBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid JSON"})
		return
	}
	if body.TransferID == nil || body.Recipient == nil || len(bytes.TrimSpace(body.Amount)) == 0 || bytes.Equal(bytes.TrimSpace(body.Amount), []byte("null")) {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "transferId, recipient, and amount are required"})
		return
	}
	amount, err := parseAmount(body.Amount)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "amount must be an integer"})
		return
	}
	notes := ""
	if body.Notes != nil {
		notes = *body.Notes
	}
	result, err := s.wallets.Transfer(r.Context(), account.ID, account.Email, wallet.Request{
		TransferID: *body.TransferID,
		Recipient:  *body.Recipient,
		Amount:     amount,
		Notes:      notes,
	})
	if err != nil {
		writeTransferError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeTransferError(w http.ResponseWriter, err error) {
	switch {
	case isTimeout(err):
		writeTimeout(w)
	case errors.Is(err, wallet.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorBody{Error: err.Error()})
	case errors.Is(err, wallet.ErrInsufficient), errors.Is(err, wallet.ErrConflict):
		writeJSON(w, http.StatusConflict, errorBody{Error: err.Error()})
	case errors.Is(err, wallet.ErrSelf),
		errors.Is(err, wallet.ErrInvalidAmount),
		errors.Is(err, wallet.ErrNotesTooLong),
		errors.Is(err, wallet.ErrTransferID),
		errors.Is(err, wallet.ErrRecipient):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not transfer"})
	}
}

func parseAmount(raw json.RawMessage) (int64, error) {
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(raw)))
	dec.UseNumber()
	var number json.Number
	if err := dec.Decode(&number); err != nil {
		return 0, err
	}
	if dec.More() {
		return 0, errors.New("amount must be an integer")
	}
	return number.Int64()
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	raw := strings.TrimSpace(header[len(prefix):])
	if raw == "" {
		return "", false
	}
	return raw, true
}

func writeTimeout(w http.ResponseWriter) {
	writeJSON(w, http.StatusGatewayTimeout, errorBody{Error: "timeout"})
}

func isTimeout(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
