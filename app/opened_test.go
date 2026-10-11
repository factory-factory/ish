package app

import (
	"strings"
	"testing"
	"time"

	"github.com/factory-factory/ish/models"
)

// stubNow fixes the time for the duration of a test.
func stubNow(t *testing.T, at time.Time) {
	t.Helper()

	old := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = old })
}

// issueOpened returns the source of an issue with the given title, opened at
// date.
func issueOpened(title, date string) string {
	return "# " + title + "\n\n[author " + date + "]\n\n- assignees: alice\n- state: open\n"
}

var today = time.Date(2026, 10, 11, 9, 0, 0, 0, time.FixedZone("", -7*60*60))

func TestOpenedFormat(t *testing.T) {
	tests := []struct {
		name string
		date string
		want string
	}{
		{"this year", "2026-03-05T10:00:00-07:00", "Mar 5"},
		{"today", "2026-10-11T08:59:23-07:00", "Oct 11"},
		{"earlier year", "2023-06-04T18:37:55-04:00", "Jun 4, 2023"},
		{"other layout", "2023-05-28 11:55:23 -0400", "May 28, 2023"},
		// The date is shown in the byline's time zone, as in the issue view,
		// even though it's a different day and year in UTC.
		{"byline time zone", "2026-01-01T00:30:00+14:00", "Jan 1"},
		{"unreadable", "someday", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := opened(models.Issue{DateCreated: tt.date}, today); got != tt.want {
				t.Errorf("opened(%q) = %q, want %q", tt.date, got, tt.want)
			}
		})
	}
}

func TestOpenedColumn(t *testing.T) {
	stubNow(t, today)

	m := newTestModel(t, map[string]string{
		"AAA": issueOpened("First", "2023-06-04T18:37:55-04:00"),
		"BBB": issueOpened("Second", "2026-10-11T08:59:23-07:00"),
		"CCC": issueOpened("Third", "someday"),
	})

	// The opened column is to the right of the assignee, and left of the
	// state.
	header := ""
	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(line, "Title") {
			header = escapes.ReplaceAllString(line, "")
			break
		}
	}

	assignee, opened, state := strings.Index(header, "Assignee"), strings.Index(header, "Opened"), strings.Index(header, "State")

	if !(assignee >= 0 && assignee < opened && opened < state) {
		t.Errorf("header %q, want Assignee, Opened, and State in that order", header)
	}

	for _, want := range []struct{ id, date string }{
		{"AAA", "Jun 4, 2023"},
		{"BBB", "Oct 11"},
	} {
		line := escapes.ReplaceAllString(row(t, m, want.id), "")

		// Dates line up under their heading.
		if got := strings.Index(line, want.date); got != opened {
			t.Errorf("row %q: %q at column %d, want %d under its heading", line, want.date, got, opened)
		}
	}

	// An unreadable date leaves the cell empty, and the issue is still listed.
	line := escapes.ReplaceAllString(row(t, m, "CCC"), "")
	if cell := strings.TrimSpace(line[opened:state]); cell != "" {
		t.Errorf("row %q: opened = %q for an unreadable date, want empty", line, cell)
	}

	// The column is as wide as its widest date.
	if got := m.table.Columns()[3]; got.Title != "Opened" || got.Width != len("Jun 4, 2023") {
		t.Errorf("column 3 = %q width %d, want Opened width %d", got.Title, got.Width, len("Jun 4, 2023"))
	}
}
