package state

import (
	"sync"
	"testing"

	"github.com/factory-factory/ish/config"
)

func TestSnapshotsAreConsistent(t *testing.T) {
	s := New(config.Config{"user": map[string]any{"email": "alice@example.com"}})
	before := s.Snapshot()

	s.SetConfig(config.Config{"user": map[string]any{"email": "bob@example.com"}})
	after := s.Snapshot()

	// An earlier snapshot keeps the state it was taken with.
	if got, _ := before.LookupString(".user.email"); got != "alice@example.com" {
		t.Errorf("earlier snapshot's email = %q, want alice@example.com", got)
	}
	if got, _ := after.LookupString(".user.email"); got != "bob@example.com" {
		t.Errorf("later snapshot's email = %q, want bob@example.com", got)
	}
}

func TestSetConfigCopies(t *testing.T) {
	cfg := config.Config{"user": map[string]any{"email": "alice@example.com"}}
	s := New(cfg)

	// Changing the configuration that was passed in doesn't change the
	// state.
	cfg["user"].(map[string]any)["email"] = "changed@example.com"

	if got, _ := s.Snapshot().LookupString(".user.email"); got != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", got)
	}
}

func TestLookupString(t *testing.T) {
	snap := New(config.Config{"name": "ish", "count": int64(2)}).Snapshot()

	if got, ok := snap.LookupString(".name"); !ok || got != "ish" {
		t.Errorf("LookupString(.name) = %q, %v, want ish", got, ok)
	}
	if _, ok := snap.LookupString(".count"); ok {
		t.Error("LookupString(.count) found a string, want none, as it's a number")
	}
	if _, ok := snap.LookupString(".missing"); ok {
		t.Error("LookupString(.missing) found a string")
	}
}

func TestEmptyState(t *testing.T) {
	var none *State

	if _, ok := none.Snapshot().Lookup(".x"); ok {
		t.Error("a nil state has a value")
	}
	if _, ok := New(nil).Snapshot().Lookup(".x"); ok {
		t.Error("a state with no configuration has a value")
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	s := New(config.Config{"n": int64(0)})

	var wg sync.WaitGroup

	for i := range 8 {
		wg.Add(2)

		go func() {
			defer wg.Done()
			s.SetConfig(config.Config{"n": int64(i), "m": int64(i)})
		}()

		go func() {
			defer wg.Done()

			// Both values come from the same snapshot, so they match.
			snap := s.Snapshot()
			n, _ := snap.Lookup(".n")
			m, ok := snap.Lookup(".m")

			if ok && n != m {
				t.Errorf("snapshot has n = %v and m = %v, want them equal", n, m)
			}
		}()
	}

	wg.Wait()
}
