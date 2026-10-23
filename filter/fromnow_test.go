package filter

import (
	"strings"
	"testing"
	"time"
)

// pacific is a time zone with daylight saving time, for calendar arithmetic.
var pacific = time.FixedZone("PST", -8*60*60)

func TestFromNow(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		la = pacific
	}

	now := time.Date(2026, 1, 31, 22, 30, 15, 500_000_000, la) // 2026-02-01T06:30:15Z

	for _, tt := range []struct {
		offset, want string
	}{
		{"0m", "2026-02-01T06:30:15Z"}, // Whole seconds, in UTC.
		{"-3h", "2026-02-01T03:30:15Z"},
		{"+90min", "2026-02-01T08:00:15Z"},
		{"2hr", "2026-02-01T08:30:15Z"},
		{"-1d", "2026-01-31T06:30:15Z"},
		{"1w", "2026-02-08T06:30:15Z"},
		// Calendar months and years in local time, clamped to the end of
		// the month: January 31 plus a month is February 28, locally.
		{"+1mo", "2026-03-01T06:30:15Z"},
		{"-2mo", "2025-12-01T06:30:15Z"},
		{"1y", "2027-02-01T06:30:15Z"},
		{"1yr", "2027-02-01T06:30:15Z"},
		// Several parts; a sign applies until the next one.
		{"-1d 6h", "2026-01-31T00:30:15Z"},
		{"-1d +6h", "2026-01-31T12:30:15Z"},
		{"  1mo   1d ", "2026-03-02T06:30:15Z"},
	} {
		if got, err := fromNow(tt.offset, now); err != nil || got != tt.want {
			t.Errorf("fromNow(%q) = %q, %v, want %q", tt.offset, got, err, tt.want)
		}
	}
}

func TestFromNowCalendar(t *testing.T) {
	for _, tt := range []struct {
		now    time.Time
		offset string
		want   string
	}{
		// January 31 plus a month is the end of February, and a leap day
		// plus a year is February 28.
		{time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC), "1mo", "2026-02-28T12:00:00Z"},
		{time.Date(2028, 2, 29, 12, 0, 0, 0, time.UTC), "1yr", "2029-02-28T12:00:00Z"},
		{time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC), "-1mo", "2026-02-28T12:00:00Z"},
		{time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC), "-17mo", "2024-12-15T12:00:00Z"},
	} {
		if got, err := fromNow(tt.offset, tt.now); err != nil || got != tt.want {
			t.Errorf("fromNow(%q) from %v = %q, %v, want %q", tt.offset, tt.now, got, err, tt.want)
		}
	}

	// A day is a calendar day across a change to daylight saving time.
	if la, err := time.LoadLocation("America/Los_Angeles"); err == nil {
		now := time.Date(2026, 3, 8, 12, 0, 0, 0, la) // After the change at 2am.

		if got, _ := fromNow("-1d", now); got != "2026-03-07T20:00:00Z" {
			t.Errorf("fromNow(-1d) across daylight saving = %q, want noon the day before, 2026-03-07T20:00:00Z", got)
		}
		if got, _ := fromNow("-24h", now); got != "2026-03-07T19:00:00Z" {
			t.Errorf("fromNow(-24h) across daylight saving = %q, want exactly 24 hours earlier", got)
		}
	}
}

func TestFromNowErrors(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, offset := range []string{"", "  ", "3", "h", "1.5h", "-3 h", "3 fortnights", "1D", "--1d", "1d,2h"} {
		if got, err := fromNow(offset, now); err == nil {
			t.Errorf("fromNow(%q) = %q, want an error", offset, got)
		}
	}

	_, err := fromNow("3fortnights", now)
	if err == nil || !strings.Contains(err.Error(), `can't read "3fortnights"`) || !strings.Contains(err.Error(), "m or min, h or hr, d, w, mo, or y or yr") {
		t.Errorf("error = %v, want one naming the part and the units", err)
	}
}

func TestResolveFromNow(t *testing.T) {
	env := Env{Lookup: lookupState, Now: time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)}

	for _, tt := range []struct {
		text, want string
	}{
		{"('-7d' :fromnow)", `"2026-10-15T12:00:00Z"`},
		{`('-\(.count)d' :fromnow)`, `"2026-10-17T12:00:00Z"`},
		{"(.missing :fromnow)", "unknown"},
	} {
		resolved, err := ResolveEnv(mustParse(t, "{a = "+tt.text+"}"), env)
		if err != nil {
			t.Errorf("%s: error = %v", tt.text, err)
			continue
		}

		data := resolved.(*AttributeExpression).Value.Data
		got := "unknown"

		if expr, ok := data.(*Expression); ok {
			v, _ := expr.Root.eval(Env{})
			got = encodeValue(v)
		}

		if got != tt.want {
			t.Errorf("%s = %s, want %s", tt.text, got, tt.want)
		}
	}

	for _, tt := range []struct {
		text, message string
	}{
		{"('soon' :fromnow)", `:fromnow can't read "soon"`},
		{"(5 :fromnow)", ":fromnow needs an offset that's a string, not a number"},
	} {
		if _, err := ResolveEnv(mustParse(t, "{a = "+tt.text+"}"), env); err == nil || !strings.Contains(err.Error(), tt.message) {
			t.Errorf("%s error = %v, want one containing %q", tt.text, err, tt.message)
		}
	}

	// The filter prints as it's written.
	if got := mustParse(t, "{@dateCreated > ('-7d' :fromnow)}").String(); got != "{@dateCreated > ('-7d' :fromnow)}" {
		t.Errorf("String() = %q", got)
	}
}

func TestFromNowUsesOneMoment(t *testing.T) {
	node := mustParse(t, "{a = ('0m' :fromnow)} and {b = ('0m' :fromnow)}")

	resolved, err := ResolveEnv(node, Env{})
	if err != nil {
		t.Fatal(err)
	}

	and := resolved.(*AndExpression)
	value := func(n Node) any {
		v, _ := n.(*AttributeExpression).Value.Data.(*Expression).Root.eval(Env{})
		return v
	}

	if a, b := value(and.LeftExpression), value(and.RightExpression); a != b {
		t.Errorf("two :fromnow in one query = %v and %v, want the same moment", a, b)
	}
}
