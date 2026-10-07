package wallet_test

import (
	"context"
	"errors"
	"testing"

	"btechdevcases/internal/wallet"
)

func TestDuplicateKeyReplayRollsBack(t *testing.T) {
	store := &commitStore{
		balances: map[string]int64{"ada": 97500, "bob": 102500},
		existing: wallet.Record{
			TransferID:  "t-1",
			SenderID:    "ada",
			RecipientID: "bob",
			Recipient:   "bob@example.com",
			Amount:      2500,
			Notes:       "lunch",
		},
	}
	got, err := wallet.New(store, nil).Transfer(context.Background(), "ada", "ada@example.com", wallet.Request{
		TransferID: "t-1",
		Recipient:  "bob@example.com",
		Amount:     2500,
		Notes:      "lunch",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != (wallet.Result{TransferID: "t-1", Recipient: "bob@example.com", Amount: 2500, Notes: "lunch"}) {
		t.Fatalf("result %+v", got)
	}
	if store.committed || store.saves != 1 {
		t.Fatalf("committed %v saves %d", store.committed, store.saves)
	}
	if store.balances["ada"] != 97500 || store.balances["bob"] != 102500 {
		t.Fatalf("balances %+v", store.balances)
	}
}

func TestDuplicateKeyConflictRollsBack(t *testing.T) {
	store := &commitStore{
		balances: map[string]int64{"ada": 97500, "bob": 102500},
		existing: wallet.Record{
			TransferID:  "t-1",
			SenderID:    "ada",
			RecipientID: "bob",
			Recipient:   "bob@example.com",
			Amount:      2500,
			Notes:       "lunch",
		},
	}
	_, err := wallet.New(store, nil).Transfer(context.Background(), "ada", "ada@example.com", wallet.Request{
		TransferID: "t-1",
		Recipient:  "bob@example.com",
		Amount:     2600,
		Notes:      "lunch",
	})
	if !errors.Is(err, wallet.ErrConflict) {
		t.Fatalf("error %v", err)
	}
	if store.committed || store.saves != 1 {
		t.Fatalf("committed %v saves %d", store.committed, store.saves)
	}
	if store.balances["ada"] != 97500 || store.balances["bob"] != 102500 {
		t.Fatalf("balances %+v", store.balances)
	}
}

type commitStore struct {
	balances  map[string]int64
	existing  wallet.Record
	committed bool
	saves     int
}

func (s *commitStore) Balance(context.Context, string) (int64, error) { return 0, nil }

func (s *commitStore) History(context.Context, string) ([]wallet.Entry, error) { return nil, nil }

func (s *commitStore) InTx(ctx context.Context, fn func(wallet.Tx) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx := &commitTx{store: s, balances: cloneBalances(s.balances), existing: s.existing}
	if err := fn(tx); err != nil {
		return err
	}
	s.balances = tx.balances
	s.committed = true
	return nil
}

func cloneBalances(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for id, balance := range in {
		out[id] = balance
	}
	return out
}

type commitTx struct {
	store    *commitStore
	balances map[string]int64
	existing wallet.Record
	reads    int
}

func (t *commitTx) FindUser(context.Context, string) (string, error) {
	return t.existing.RecipientID, nil
}

func (t *commitTx) LockBalances(context.Context, []string) (map[string]int64, error) {
	return t.balances, nil
}

func (t *commitTx) TransferByID(context.Context, string, string) (wallet.Record, bool, error) {
	t.reads++
	if t.reads == 1 {
		return wallet.Record{}, false, nil
	}
	return t.existing, true, nil
}

func (t *commitTx) SaveBalances(context.Context, map[string]int64) error { return nil }

func (t *commitTx) SaveTransfer(context.Context, wallet.Record) error {
	t.store.saves++
	return wallet.ErrDuplicateTransfer
}
