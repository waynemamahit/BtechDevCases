package user_test

import (
	"errors"
	"testing"

	"btechdevcases/internal/user"
)

func TestRegisterValidation(t *testing.T) {
	email := "ada@example.com"
	password := "s3cret"
	other := "other-secret"
	empty := ""
	notAnEmail := "not-an-email"

	cases := []struct {
		name string
		in   user.Registration
		want error
	}{
		{
			name: "missing email",
			in:   user.Registration{Password: &password, ConfirmPassword: &password},
			want: user.ErrMissingEmail,
		},
		{
			name: "missing password",
			in:   user.Registration{Email: &email, ConfirmPassword: &password},
			want: user.ErrMissingPassword,
		},
		{
			name: "missing confirmPassword",
			in:   user.Registration{Email: &email, Password: &password},
			want: user.ErrMissingConfirmPassword,
		},
		{
			name: "email not-an-email",
			in:   user.Registration{Email: &notAnEmail, Password: &password, ConfirmPassword: &password},
			want: user.ErrInvalidEmail,
		},
		{
			name: "empty password",
			in:   user.Registration{Email: &email, Password: &empty, ConfirmPassword: &empty},
			want: user.ErrEmptyPassword,
		},
		{
			name: "confirmPassword differs",
			in:   user.Registration{Email: &email, Password: &password, ConfirmPassword: &other},
			want: user.ErrPasswordMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := user.Validate(tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestValidateAcceptsExactEmailString(t *testing.T) {
	email := "Ada@Example.com"
	password := "s3cret"
	if err := user.Validate(user.Registration{
		Email:           &email,
		Password:        &password,
		ConfirmPassword: &password,
	}); err != nil {
		t.Fatal(err)
	}
}
