package app

import (
	"strings"
	"testing"

	"github.com/factory-factory/ish/config"
	"github.com/factory-factory/ish/state"
	"github.com/factory-factory/ish/store"
)

func TestFilterWithLookup(t *testing.T) {
	m := menuModel(t)
	m.SetState(state.New(config.Config{
		"user":  map[string]any{"email": "alice@example.com"},
		"team":  map[string]any{"lead": map[string]any{"x": "y"}},
		"other": "nobody@example.com",
	}))

	// The lookup is resolved from the state.
	m = applyFilter(t, m, "{assignees = .user.email} and {state != 'closed'}")

	if m.filter.err != nil || ids(m) != "AAA BBB EEE" {
		t.Errorf("err, listed = %v, %q, want alice's unclosed issues", m.filter.err, ids(m))
	}

	// The filter shows the lookup, in its own style.
	if view := plainView(m); !strings.Contains(view, ".user.email") {
		t.Errorf("View() = %q, want the lookup in the filter", view)
	}

	// A value that can't be compared is shown as an error, and the filter
	// isn't applied.
	m = applyFilter(t, m, "{assignees = .team.lead}")

	if !m.filter.editing || m.filter.err == nil || !strings.Contains(m.filter.err.Error(), ".team.lead is a table") {
		t.Errorf("editing, err = %v, %v, want the type error", m.filter.editing, m.filter.err)
	}
	if got := ids(m); got != "AAA BBB EEE" {
		t.Errorf("listed %q, want the list unchanged", got)
	}
}

func TestFilterWithMissingLookup(t *testing.T) {
	m := menuModel(t)
	m.SetState(state.New(nil))

	// A missing value can't be compared, so nothing matches, but an or
	// still matches its other side.
	m = applyFilter(t, m, "{assignees = .user.email}")

	if m.filter.err != nil || ids(m) != "" {
		t.Errorf("err, listed = %v, %q, want nothing listed", m.filter.err, ids(m))
	}

	m = applyFilter(t, m, "{assignees = .user.email} or {assignees = 'bob'}")

	if got := ids(m); got != "BBB DDD" {
		t.Errorf("listed %q, want bob's issues", got)
	}
}

func TestFilterLookupsReadCurrentState(t *testing.T) {
	m := menuModel(t)
	st := state.New(config.Config{"who": "bob"})
	m.SetState(st)
	m = applyFilter(t, m, "{assignees = .who}")

	if got := ids(m); got != "BBB DDD" {
		t.Fatalf("listed %q, want bob's issues", got)
	}

	// Reloading resolves the filter again, from the state as it is now.
	st.SetConfig(config.Config{"who": "alice@example.com"})
	m.reload()

	if got := ids(m); got != "AAA BBB CCC EEE" {
		t.Errorf("listed %q after the state changed, want alice's issues", got)
	}
}

func TestFilterWithExpression(t *testing.T) {
	m := menuModel(t)
	m = applyFilter(t, m, "{assignees = ('ALICE@EXAMPLE.COM' :lower)} and {state != 'closed'}")

	if m.filter.err != nil || ids(m) != "AAA BBB EEE" {
		t.Errorf("err, listed = %v, %q, want alice's unclosed issues", m.filter.err, ids(m))
	}

	// An expression that can't be evaluated is shown as an error, and the
	// filter isn't applied.
	m = applyFilter(t, m, "{assignees = ('a' + 1)}")

	if !m.filter.editing || m.filter.err == nil || !strings.Contains(m.filter.err.Error(), "+ needs two numbers") {
		t.Errorf("editing, err = %v, %v, want the expression's error", m.filter.editing, m.filter.err)
	}
}

func TestFilterWithUser(t *testing.T) {
	m := menuModel(t)
	m = addUsers(t, m, store.User{Name: "Alice Liddell", Email: "alice@example.com"})
	m = applyFilter(t, m, "{assignees = ('@alice liddell' :user)} and {state != 'closed'}")

	if m.filter.err != nil || ids(m) != "AAA BBB EEE" {
		t.Errorf("err, listed = %v, %q, want alice's unclosed issues", m.filter.err, ids(m))
	}
}
