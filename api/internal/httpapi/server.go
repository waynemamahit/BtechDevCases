package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"btechdevcases/internal/token"
	"btechdevcases/internal/user"
)

type Server struct {
	users  user.Directory
	tokens *token.Issuer
	origin string
}

func NewServer(users user.Directory, tokens *token.Issuer, origin string) *Server {
	return &Server{users: users, tokens: tokens, origin: origin}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", s.handleRegister)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /me", s.handleMe)
	return s.withCORS(mux)
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

type identityResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
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
		writeRegisterError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, accountResponse{ID: account.ID, Email: account.Email})
}

func writeRegisterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrDuplicateEmail):
		writeJSON(w, http.StatusConflict, errorBody{Error: err.Error()})
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
	account, err := s.users.Authenticate(r.Context(), in.Email, in.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorBody{Error: "invalid credentials"})
		return
	}
	signed, err := s.tokens.Sign(account.ID, account.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not sign token"})
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{Token: signed})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
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
	signed, err := s.tokens.Sign(claims.Subject, claims.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "could not sign token"})
		return
	}
	writeJSON(w, http.StatusOK, identityResponse{
		ID:    claims.Subject,
		Email: claims.Email,
		Token: signed,
	})
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
