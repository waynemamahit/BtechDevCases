package httpapi_test

import (
	"net/http"
	"testing"
)

func TestRegisterAcceptedOmitsSecrets(t *testing.T) {
	api := newAPI(t)
	rec := postJSON(api.handler, "/register", `{"email":"ada@example.com","password":"s3cret","confirmPassword":"s3cret"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	decodeJSON(t, rec, &body)
	if len(body) != 2 {
		t.Fatalf("body keys = %v", body)
	}
	if body["email"] != "ada@example.com" {
		t.Fatalf("email = %v", body["email"])
	}
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}
	if _, ok := body["token"]; ok {
		t.Fatal("response included a token")
	}
	if _, ok := body["password"]; ok {
		t.Fatal("response included a password")
	}
	if !api.store.PasswordMatches("ada@example.com", "s3cret") {
		t.Fatal("stored account does not match the submitted password")
	}
}

func TestRegisterValidationFailures(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "missing email", body: `{"password":"s3cret","confirmPassword":"s3cret"}`},
		{name: "missing password", body: `{"email":"ada@example.com","confirmPassword":"s3cret"}`},
		{name: "missing confirmPassword", body: `{"email":"ada@example.com","password":"s3cret"}`},
		{name: "not-an-email", body: `{"email":"not-an-email","password":"s3cret","confirmPassword":"s3cret"}`},
		{name: "empty password", body: `{"email":"ada@example.com","password":"","confirmPassword":""}`},
		{name: "confirmPassword differs", body: `{"email":"ada@example.com","password":"s3cret","confirmPassword":"other"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := newAPI(t)
			rec := postJSON(api.handler, "/register", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			if _, ok := api.store.Lookup("ada@example.com"); ok {
				t.Fatal("account was created")
			}
			if _, ok := api.store.Lookup("not-an-email"); ok {
				t.Fatal("account was created")
			}
		})
	}
}

func TestRegisterDuplicateKeepsOriginalPassword(t *testing.T) {
	api := newAPI(t)
	first := postJSON(api.handler, "/register", `{"email":"ada@example.com","password":"first-secret","confirmPassword":"first-secret"}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first status %d body %s", first.Code, first.Body.String())
	}

	second := postJSON(api.handler, "/register", `{"email":"ada@example.com","password":"second-secret","confirmPassword":"second-secret"}`)
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate status %d body %s", second.Code, second.Body.String())
	}
	if !api.store.PasswordMatches("ada@example.com", "first-secret") {
		t.Fatal("original password no longer matches")
	}
	if api.store.PasswordMatches("ada@example.com", "second-secret") {
		t.Fatal("duplicate registration replaced the original password")
	}
}
