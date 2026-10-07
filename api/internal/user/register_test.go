package user_test

import (
	"errors"
	"testing"

	"btechdevcases/internal/user"
)

func TestValidate(t *testing.T) {
	email := "ada@example.com"
	password := "secret"
	other := "other"
	empty := ""
	notAnEmail := "not-an-email"
	named := "Ada <ada@example.com>"
	mixed := "Ada@example.com"

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
			name: "empty email",
			in:   user.Registration{Email: &empty, Password: &password, ConfirmPassword: &password},
			want: user.ErrInvalidEmail,
		},
		{
			name: "email with display name",
			in:   user.Registration{Email: &named, Password: &password, ConfirmPassword: &password},
			want: user.ErrInvalidEmail,
		},
		{
			name: "empty password before confirmation",
			in:   user.Registration{Email: &email, Password: &empty, ConfirmPassword: &empty},
			want: user.ErrEmptyPassword,
		},
		{
			name: "empty password differs from confirmation",
			in:   user.Registration{Email: &email, Password: &empty, ConfirmPassword: &other},
			want: user.ErrEmptyPassword,
		},
		{
			name: "confirmPassword differs",
			in:   user.Registration{Email: &email, Password: &password, ConfirmPassword: &other},
			want: user.ErrPasswordMismatch,
		},
		{
			name: "exact address",
			in:   user.Registration{Email: &email, Password: &password, ConfirmPassword: &password},
		},
		{
			name: "case preserved address",
			in:   user.Registration{Email: &mixed, Password: &password, ConfirmPassword: &password},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := user.Validate(tc.in)
			if tc.want == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}
