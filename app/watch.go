package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/models"
	"github.com/factory-factory/ish/parse"
	"github.com/factory-factory/ish/store"
	"github.com/factory-factory/ish/styles"
)

// Watch sets the channel that receives a value when the issue files change
// outside ish, as watch.Watcher's Changes does. Each time, the metadata
// database is brought up to date with the files, the list shows the changes,
// and so does the open issue. If the open issue has unsaved edits, a dialog
// asks whether to discard them and load the changes, or keep them and
// overwrite the changes.
func (m *Model) Watch(changes <-chan struct{}) {
	m.changes = changes
}

// changedMsg is sent when the issue files have changed.
type changedMsg struct{}

// waitForChanges returns a command that waits for the issue files to change,
// or nil if they aren't watched.
func (m Model) waitForChanges() tea.Cmd {
	changes := m.changes

	if changes == nil {
		return nil
	}

	return func() tea.Msg {
		if _, ok := <-changes; !ok {
			return nil
		}

		return changedMsg{}
	}
}

// syncFiles brings the metadata database up to date with the issue files,
// then shows the changes in the list, or in the open issue if they're to its
// file, and waits for the next change.
func (m Model) syncFiles() (tea.Model, tea.Cmd) {
	wait := m.waitForChanges()

	// The database is synced here, rather than in a command, so it's never
	// written by two goroutines at once.
	result, err := store.Sync(m.dir)

	if err != nil {
		err = fmt.Errorf("couldn't update the metadata database with the changes to the issue files: %w", err)

		if issue, ok := m.issue.(models.Issue); ok {
			issue.SetNotice(err.Error())
			m.issue = issue
		} else {
			m.err = err
			m.resize()
		}

		return m, wait
	}

	issue, ok := m.issue.(models.Issue)

	if !ok {
		m.reload()
		return m, wait
	}

	id := m.openId

	switch {
	case slices.Contains(result.Removed, id):
		if issue.Unsaved() {
			m.setConflict(true)
			return m, wait
		}

		return m.closeDeleted(), wait

	case failedFile(result, id) != nil:
		// The last version that could be read stays open, so a file that's
		// being written, or has merge conflicts, doesn't close it.
		issue.SetNotice(fmt.Sprintf("Couldn't load the changes made to %s outside ish: %v", id+".md", failedFile(result, id)))
		m.issue = issue

	// A file that couldn't be read is added back once it can be.
	case slices.Contains(result.Updated, id) || slices.Contains(result.Added, id):
		if issue.Unsaved() {
			m.setConflict(false)
			return m, wait
		}

		m.reloadOpen()
	}

	return m, wait
}

// failedFile returns the error indexing the file of the issue with the given
// ID, if result has one.
func failedFile(result *store.SyncResult, id string) error {
	for _, failure := range result.Failed {
		if filepath.Base(failure.Path) == id+".md" {
			return failure.Err
		}
	}

	return nil
}

// reloadOpen shows the open issue as its file is now, selected and scrolled
// as it was. If the file can't be loaded, the issue stays as it was, with
// the error shown.
func (m *Model) reloadOpen() {
	old := m.issue.(models.Issue)

	loaded, err := parse.LoadIssue(m.dir, m.openId)
	if err != nil {
		old.SetNotice(fmt.Sprintf("Couldn't load the changes made to %s outside ish: %v", m.openId+".md", err))
		m.issue = old

		return
	}

	issue := m.prepare(*loaded)
	issue.Follow(old)
	m.issue = issue
}

// closeDeleted closes the open issue, whose file was deleted, and shows the
// list, saying why. An issue ish was started with is closed too, since it's
// gone.
func (m Model) closeDeleted() Model {
	id := m.openId

	m.issue = nil
	m.openId = ""
	m.single = false
	m.reload()
	m.err = fmt.Errorf("%s was deleted outside ish", id)
	m.resize()

	return m
}

// conflict is a change made outside ish to the open issue's file while the
// issue has unsaved edits.
type conflict struct {
	// deleted reports whether the file was deleted, rather than changed.
	deleted bool

	// keep reports whether the button to keep the edits is selected, rather
	// than the one to discard them.
	keep bool
}

// setConflict asks which to keep, the open issue's edits or the change to
// its file, deleted or not. If it's already asking, the selected answer
// stays selected.
func (m *Model) setConflict(deleted bool) {
	if m.conflict == nil {
		// Keeping the edits is selected first, since they're what's being
		// worked on.
		m.conflict = &conflict{keep: true}
	}

	m.conflict.deleted = deleted
}

// conflictKeys are the key bindings for the conflict dialog.
var conflictKeys = struct {
	Move   key.Binding
	Choose key.Binding
}{
	Move: key.NewBinding(
		key.WithKeys("left", "right", "h", "l", "tab", "shift+tab"),
		key.WithHelp("←/→", "move"),
	),
	Choose: key.NewBinding(
		key.WithKeys("enter", "space"),
		key.WithHelp("enter", "choose"),
	),
}

// updateConflict handles a key press while the conflict dialog is shown.
// Keys other than those to move between its buttons and choose one are
// ignored, so the question is answered before anything else is done.
func (m Model) updateConflict(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, conflictKeys.Move):
		m.conflict.keep = !m.conflict.keep

	case key.Matches(msg, conflictKeys.Choose):
		return m.resolveConflict()
	}

	return m, nil
}

// resolveConflict answers the conflict dialog with the selected button. Keeping
// the edits saves the issue now, over the file, or in place of the deleted
// one, and the field being edited, if any, goes on being edited. Discarding
// them shows the file as it is now, or the list if it was deleted.
func (m Model) resolveConflict() (tea.Model, tea.Cmd) {
	c := *m.conflict
	m.conflict = nil

	if c.keep {
		issue := m.issue.(models.Issue)
		issue.SetNotice("")

		// If it can't be saved, the issue shows the error, as it does when
		// it's closed.
		_ = issue.Save()
		m.issue = issue

		return m, nil
	}

	if c.deleted {
		return m.closeDeleted(), nil
	}

	m.reloadOpen()

	return m, nil
}

// discardLabel and keepLabel are the conflict dialog's buttons' labels.
const (
	discardLabel = "Discard my edits"
	keepLabel    = "Keep my edits"
)

// renderConflict renders the conflict dialog, at most width wide.
func (m Model) renderConflict(width int) string {
	file := m.openId + ".md"

	heading := file + " was changed outside ish."
	question := "Discard your edits and load the outside changes, or keep your edits and overwrite the outside changes?"

	if m.conflict.deleted {
		heading = file + " was deleted outside ish."
		question = "Discard your edits and close the issue, or keep your edits and save the issue again?"
	}

	discard, keep := styles.Button, styles.ButtonSelected
	if !m.conflict.keep {
		discard, keep = styles.ButtonCancelSelected, styles.Button
	}

	padding := strings.Repeat(" ", buttonPadding)
	buttons := discard.Render(padding+discardLabel+padding) +
		strings.Repeat(" ", buttonGap) +
		keep.Render(padding+keepLabel+padding)

	inner := min(max(lipgloss.Width(buttons), 60), max(width-styles.Dialog.GetHorizontalFrameSize(), lipgloss.Width(buttons)))
	text := lipgloss.NewStyle().Width(inner)

	return styles.Dialog.Render(lipgloss.JoinVertical(lipgloss.Left,
		text.Render(styles.Error.Render(heading)),
		"",
		text.Render(question),
		"",
		lipgloss.PlaceHorizontal(inner, lipgloss.Right, buttons),
		"",
		lipgloss.PlaceHorizontal(inner, lipgloss.Right, styles.Byline.Render("←/→ move • enter choose")),
	))
}

// overlayConflict draws the conflict dialog over content, the open issue's
// view, in the middle of the terminal.
func (m Model) overlayConflict(content string) string {
	dialog := m.renderConflict(m.width)

	x := max((m.width-lipgloss.Width(dialog))/2, 0)
	y := max((m.height-lipgloss.Height(dialog))/2, 0)

	return lipgloss.NewCompositor(
		lipgloss.NewLayer(content),
		lipgloss.NewLayer(dialog).X(x).Y(y).Z(1),
	).Render()
}
