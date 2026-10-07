package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type walletView struct {
	Balance   int64 `json:"balance"`
	Transfers []struct {
		TransferID   string `json:"transferId"`
		Direction    string `json:"direction"`
		Counterparty string `json:"counterparty"`
		Amount       int64  `json:"amount"`
		Notes        string `json:"notes"`
	} `json:"transfers"`
}

type transferView struct {
	TransferID string `json:"transferId"`
	Recipient  string `json:"recipient"`
	Amount     int64  `json:"amount"`
	Notes      string `json:"notes"`
}

func TestWallet(t *testing.T) {
	t.Run("opening balance", func(t *testing.T) {
		api := newAPI(t)
		registerAccount(t, api.handler, "ada@example.com", "s3cret")
		token := loginToken(t, api.handler, "ada@example.com", "s3cret")
		rec := getAuth(api.handler, "/wallet", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		var raw struct {
			Balance   int64           `json:"balance"`
			Transfers json.RawMessage `json:"transfers"`
		}
		decodeJSON(t, rec, &raw)
		if raw.Balance != 100000 {
			t.Fatalf("balance %d", raw.Balance)
		}
		if string(raw.Transfers) != "[]" {
			t.Fatalf("transfers %s", raw.Transfers)
		}
	})

	t.Run("both balances change once", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		registerAccount(t, api.handler, "cara@example.com", "s3cret")
		cara := loginToken(t, api.handler, "cara@example.com", "s3cret")

		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-lunch", "bob@example.com", 2500, "lunch"))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		var sent transferView
		decodeJSON(t, rec, &sent)
		if sent.TransferID != "t-lunch" || sent.Recipient != "bob@example.com" || sent.Amount != 2500 || sent.Notes != "lunch" {
			t.Fatalf("transfer %+v", sent)
		}

		adaWallet := mustWallet(t, api.handler, ada)
		bobWallet := mustWallet(t, api.handler, bob)
		caraWallet := mustWallet(t, api.handler, cara)
		if adaWallet.Balance != 97500 || bobWallet.Balance != 102500 || caraWallet.Balance != 100000 {
			t.Fatalf("balances ada %d bob %d cara %d", adaWallet.Balance, bobWallet.Balance, caraWallet.Balance)
		}
		if len(adaWallet.Transfers) != 1 || len(bobWallet.Transfers) != 1 || len(caraWallet.Transfers) != 0 {
			t.Fatalf("history ada %d bob %d cara %d", len(adaWallet.Transfers), len(bobWallet.Transfers), len(caraWallet.Transfers))
		}
		if adaWallet.Transfers[0].Direction != "sent" || adaWallet.Transfers[0].Counterparty != "bob@example.com" || adaWallet.Transfers[0].Notes != "lunch" {
			t.Fatalf("ada history %+v", adaWallet.Transfers[0])
		}
		if bobWallet.Transfers[0].Direction != "received" || bobWallet.Transfers[0].Counterparty != "ada@example.com" || bobWallet.Transfers[0].Amount != 2500 || bobWallet.Transfers[0].Notes != "lunch" {
			t.Fatalf("bob history %+v", bobWallet.Transfers[0])
		}
	})

	t.Run("empty notes", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-empty", "bob@example.com", 100, ""))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		adaWallet := mustWallet(t, api.handler, ada)
		bobWallet := mustWallet(t, api.handler, bob)
		if len(adaWallet.Transfers) != 1 || adaWallet.Transfers[0].Notes != "" || bobWallet.Transfers[0].Notes != "" {
			t.Fatalf("ada %+v bob %+v", adaWallet.Transfers, bobWallet.Transfers)
		}
	})

	t.Run("notes too long", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-long", "bob@example.com", 100, strings.Repeat("n", 201)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 100000, 100000)
	})

	t.Run("zero amount", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-zero", "bob@example.com", 0, ""))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 100000, 100000)
	})

	t.Run("negative amount", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-neg", "bob@example.com", -1, ""))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 100000, 100000)
	})

	t.Run("unknown recipient", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-unknown", "nobody@example.com", 100, ""))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 100000, 100000)
	})

	t.Run("self transfer", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-self", "ada@example.com", 100, ""))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 100000, 100000)
	})

	t.Run("insufficient funds", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		rec := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-over", "bob@example.com", 100001, ""))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 100000, 100000)
	})

	t.Run("same transfer id", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		body := transferJSON(t, "t-same", "bob@example.com", 2500, "rent")
		first := postAuth(api.handler, "/transfers", ada, body)
		if first.Code != http.StatusOK {
			t.Fatalf("first status %d body %s", first.Code, first.Body.String())
		}
		second := postAuth(api.handler, "/transfers", ada, body)
		if second.Code != http.StatusOK {
			t.Fatalf("second status %d body %s", second.Code, second.Body.String())
		}
		var original, replay transferView
		decodeJSON(t, first, &original)
		decodeJSON(t, second, &replay)
		if replay != original {
			t.Fatalf("replay %+v original %+v", replay, original)
		}
		assertBalances(t, api.handler, ada, bob, 97500, 102500)
		if len(mustWallet(t, api.handler, ada).Transfers) != 1 {
			t.Fatal("replay inserted another transfer")
		}
	})

	t.Run("same id different amount", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		first := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-diff", "bob@example.com", 2500, "rent"))
		if first.Code != http.StatusOK {
			t.Fatalf("first status %d body %s", first.Code, first.Body.String())
		}
		second := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-diff", "bob@example.com", 2600, "rent"))
		if second.Code != http.StatusConflict {
			t.Fatalf("second status %d body %s", second.Code, second.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 97500, 102500)
	})

	t.Run("same id different recipient", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		registerAccount(t, api.handler, "cara@example.com", "s3cret")
		first := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-who", "bob@example.com", 2500, "rent"))
		if first.Code != http.StatusOK {
			t.Fatalf("first status %d body %s", first.Code, first.Body.String())
		}
		second := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-who", "cara@example.com", 2500, "rent"))
		if second.Code != http.StatusConflict {
			t.Fatalf("second status %d body %s", second.Code, second.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 97500, 102500)
		history := mustWallet(t, api.handler, ada)
		if len(history.Transfers) != 1 || history.Transfers[0].Counterparty != "bob@example.com" || history.Transfers[0].Notes != "rent" {
			t.Fatalf("history %+v", history.Transfers)
		}
	})

	t.Run("same id different notes", func(t *testing.T) {
		api := newAPI(t)
		_, ada, bob := registerPair(t, api)
		first := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-notes", "bob@example.com", 2500, "rent"))
		if first.Code != http.StatusOK {
			t.Fatalf("first status %d body %s", first.Code, first.Body.String())
		}
		second := postAuth(api.handler, "/transfers", ada, transferJSON(t, "t-notes", "bob@example.com", 2500, "food"))
		if second.Code != http.StatusConflict {
			t.Fatalf("second status %d body %s", second.Code, second.Body.String())
		}
		assertBalances(t, api.handler, ada, bob, 97500, 102500)
		history := mustWallet(t, api.handler, ada)
		if len(history.Transfers) != 1 || history.Transfers[0].Notes != "rent" {
			t.Fatalf("history %+v", history.Transfers)
		}
	})

	t.Run("wallet read extends session", func(t *testing.T) {
		api := newAPI(t)
		id := registerAccount(t, api.handler, "ada@example.com", "s3cret")
		start := api.clock.Now()
		token := loginToken(t, api.handler, "ada@example.com", "s3cret")

		api.clock.Set(start.Add(14 * time.Minute))
		view := mustWallet(t, api.handler, token)
		if view.Balance != 100000 || len(view.Transfers) != 0 {
			t.Fatalf("wallet %+v", view)
		}
		assertActivity(t, api, id, start.Add(14*time.Minute))

		api.clock.Set(start.Add(28 * time.Minute))
		rec := getMe(api.handler, token)
		if rec.Code != http.StatusOK || rec.Body.String() != welcomeAda {
			t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("failed transfer does not extend session", func(t *testing.T) {
		api := newAPI(t)
		id := registerAccount(t, api.handler, "ada@example.com", "s3cret")
		registerAccount(t, api.handler, "bob@example.com", "s3cret")
		start := api.clock.Now()
		token := loginToken(t, api.handler, "ada@example.com", "s3cret")

		api.clock.Set(start.Add(10 * time.Minute))
		rec := postAuth(api.handler, "/transfers", token, transferJSON(t, "t-fail", "bob@example.com", 100001, ""))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
		}
		assertActivity(t, api, id, start)

		api.clock.Set(start.Add(15 * time.Minute))
		later := getMe(api.handler, token)
		if later.Code != http.StatusUnauthorized || later.Body.String() == welcomeAda {
			t.Fatalf("status %d body %q", later.Code, later.Body.String())
		}
		assertActivity(t, api, id, start)
	})
}

func registerPair(t *testing.T, api apiFixture) (id, adaToken, bobToken string) {
	t.Helper()
	id = registerAccount(t, api.handler, "ada@example.com", "s3cret")
	registerAccount(t, api.handler, "bob@example.com", "s3cret")
	adaToken = loginToken(t, api.handler, "ada@example.com", "s3cret")
	bobToken = loginToken(t, api.handler, "bob@example.com", "s3cret")
	return id, adaToken, bobToken
}

func transferJSON(t *testing.T, id, recipient string, amount int64, notes string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"transferId": id,
		"recipient":  recipient,
		"amount":     amount,
		"notes":      notes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func mustWallet(t *testing.T, handler http.Handler, token string) walletView {
	t.Helper()
	rec := getAuth(handler, "/wallet", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet status %d body %s", rec.Code, rec.Body.String())
	}
	var body walletView
	decodeJSON(t, rec, &body)
	return body
}

func assertBalances(t *testing.T, handler http.Handler, adaToken, bobToken string, adaWant, bobWant int64) {
	t.Helper()
	ada := mustWallet(t, handler, adaToken)
	bob := mustWallet(t, handler, bobToken)
	if ada.Balance != adaWant || bob.Balance != bobWant {
		t.Fatalf("balances ada %d bob %d", ada.Balance, bob.Balance)
	}
	if adaWant == 100000 && len(ada.Transfers) != 0 {
		t.Fatalf("ada history %d", len(ada.Transfers))
	}
}
