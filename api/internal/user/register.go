package user

import (
	"errors"
	"net/mail"
)

var (
	ErrMissingEmail           = errors.New("email is required")
	ErrMissingPassword        = errors.New("password is required")
	ErrMissingConfirmPassword = errors.New("confirmPassword is required")
	ErrInvalidEmail           = errors.New("email is not a valid address")
	ErrEmptyPassword          = errors.New("password must be non-empty")
	ErrPasswordMismatch       = errors.New("confirmPassword does not match password")
)

type Registration struct {
	Email           *string `json:"email"`
	Password        *string `json:"password"`
	ConfirmPassword *string `json:"confirmPassword"`
}

func Validate(in Registration) error {
	if in.Email == nil {
		return ErrMissingEmail
	}
	if in.Password == nil {
		return ErrMissingPassword
	}
	if in.ConfirmPassword == nil {
		return ErrMissingConfirmPassword
	}
	addr, err := mail.ParseAddress(*in.Email)
	if err != nil || addr.Name != "" || addr.Address != *in.Email {
		return ErrInvalidEmail
	}
	if *in.Password == "" {
		return ErrEmptyPassword
	}
	if *in.ConfirmPassword != *in.Password {
		return ErrPasswordMismatch
	}
	return nil
}
