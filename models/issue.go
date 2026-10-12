package models

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/factory-factory/ish/styles"
)

type Issue struct {
	Id          string
	DateCreated string
	Author      string
	Title       string

	Metadata []Tag

	Conversation []Card

	selectedComment int
	cursor          int

	// width and height are the terminal dimensions.
	width  int
	height int

	// viewport scrolls the cards below the pinned header, and spans records
	// where each card sits in the viewport content.
	viewport viewport.Model
	spans    []span

	help  help.Model
	ready bool

	// closable reports whether the back key closes the issue. See
	// SetClosable.
	closable bool

	// light reports whether the terminal has a light background.
	light bool

	// shown reports whether the issue has been sized to the terminal, which
	// happens before it's first shown.
	shown bool

	// editing reports whether the field under the cursor is being edited.
	// Tags and cards hold their own editors, and the title is edited in
	// titleInput.
	editing    bool
	titleInput textinput.Model

	// save writes the issue when it's closed or the program quits, if dirty
	// reports that it has unsaved changes. See SetSave.
	save  SaveFunc
	dirty bool

	// saveErr is the error from the last attempt to save the issue. While
	// it's set, closing the issue or quitting discards unsaved changes.
	saveErr error

	// author is the author of new comments. See SetAuthor.
	author string

	// newComment reports whether the card being edited is a comment that was
	// just added, which is removed if its first edit is cancelled or empty.
	newComment bool

	// newTag reports whether the tag being edited was just added. Its key is
	// edited first, then its values, and it's removed if it's left without
	// either.
	newTag bool
}

// addTag appends a blank metadata tag, selects it, and starts editing its
// key.
func (ish *Issue) addTag() tea.Cmd {
	ish.Metadata = append(ish.Metadata, Tag{})
	ish.setCursor(1 + len(ish.Metadata))

	cmd := ish.Metadata[ish.tagIndex()].startKeyEdit(ish.headerWidth(), !ish.light)
	ish.editing = true
	ish.newTag = true
	ish.saveErr = nil
	// The header has a new line, and the help footer changes while editing.
	ish.resize()

	return cmd
}

// saveNewTag handles saving while a new tag is being edited, reporting
// whether it was handled. Saving the key moves on to editing the values,
// unless the key is empty, which removes the tag. Saving without values
// keeps editing them. Saving with values isn't handled here, and is saved as
// any other tag edit.
func (ish *Issue) saveNewTag() (tea.Cmd, bool) {
	i := ish.tagIndex()

	if !ish.newTag || i < 0 {
		return nil, false
	}

	tag := &ish.Metadata[i]

	switch {
	case tag.editingKey && tag.editedKey() == "":
		ish.stopEdit(false)
		return nil, true

	case tag.editingKey:
		tag.commitKey()
		return tag.startEdit(ish.headerWidth(), !ish.light), true

	case len(splitValues(tag.input.Value())) == 0:
		return nil, true
	}

	return nil, false
}

// removeTag removes the metadata tag at index i, then selects the tag that
// took its place, the tag before it if it was the last, or the title if no
// tags are left.
func (ish *Issue) removeTag(i int) {
	ish.Metadata = slices.Delete(ish.Metadata, i, i+1)
	ish.cursor = 0

	if len(ish.Metadata) > 0 {
		ish.setCursor(2 + min(i, len(ish.Metadata)-1))
	} else {
		ish.setCursor(1)
	}

	ish.resize()
}

// confirmingDelete reports whether the selected tag is highlighted, waiting
// for the user to confirm that it should be deleted.
func (ish Issue) confirmingDelete() bool {
	i := ish.tagIndex()
	return i >= 0 && ish.Metadata[i].deleting
}

// updateDelete handles a key press while confirming that the selected tag
// should be deleted: the delete key deletes it, and any other key cancels.
func (ish *Issue) updateDelete(msg tea.KeyPressMsg) {
	i := ish.tagIndex()

	if key.Matches(msg, deleteKeys.Confirm) {
		ish.removeTag(i)
		ish.dirty = true
		// After a failed save, a new change may fix the problem, so the next
		// exit tries to save again.
		ish.saveErr = nil
		return
	}

	ish.Metadata[i].deleting = false
}

// now returns the current time. Tests replace it.
var now = time.Now

// SetAuthor sets the author of comments added to the issue. Without one,
// comments can't be added.
func (ish *Issue) SetAuthor(author string) {
	ish.author = author
}

// addComment appends an empty comment by the issue's author, selects it, and
// starts editing it.
func (ish *Issue) addComment() tea.Cmd {
	content := ""

	ish.Conversation = append(ish.Conversation, Card{
		Author:      ish.author,
		DateCreated: now().Format(time.RFC3339),
		Content:     &content,
	})

	ish.syncViewport()
	ish.setCursor(ish.firstCard() + len(ish.Conversation) - 1)
	cmd := ish.startEdit()
	ish.newComment = true

	return cmd
}

// removeNewComment removes the comment that was just added, and selects the
// comment before it, if any.
func (ish *Issue) removeNewComment() {
	ish.Conversation = ish.Conversation[:len(ish.Conversation)-1]
	ish.cursor = 0

	if len(ish.Conversation) > 0 {
		ish.setCursor(ish.firstCard() + len(ish.Conversation) - 1)
	}
}

// SaveFunc writes an issue to storage.
type SaveFunc func(Issue) error

// SetSave sets the function used to save the issue's changes when it's closed
// or the program quits. Without one, changes are discarded.
func (ish *Issue) SetSave(save SaveFunc) {
	ish.save = save
}

// exit saves any unsaved changes before returning cmd, which closes the issue
// or quits. If the changes can't be saved, the error is shown and the issue
// stays open; exiting again discards the changes.
func (ish *Issue) exit(cmd tea.Cmd) tea.Cmd {
	if ish.dirty && ish.save != nil && ish.saveErr == nil {
		if err := ish.save(*ish); err != nil {
			ish.saveErr = err
			ish.resize()
			return nil
		}

		ish.dirty = false
	}

	return cmd
}

// CloseMsg is sent when the user closes an issue with the back key.
type CloseMsg struct{}

// SetClosable sets whether the back key closes the issue by sending a
// CloseMsg, for when the issue is shown by a model that can return to a
// previous view.
func (ish *Issue) SetClosable(closable bool) {
	ish.closable = closable
}

// keyMap returns the issue's key bindings, with the back key enabled if the
// issue can be closed.
func (ish Issue) keyMap() keyMap {
	k := keys
	k.Edit.SetEnabled(ish.cursor > 0)
	k.Comment.SetEnabled(ish.author != "")
	k.DeleteTag.SetEnabled(ish.tagIndex() >= 0)
	k.Back.SetEnabled(ish.closable)

	if i := ish.cardIndex(); i >= 0 {
		k.Todos.SetEnabled(len(ish.Conversation[i].todos()) > 0)
	}

	return k
}

// selectingTodo reports whether the todo items of the card under the cursor
// are being selected.
func (ish Issue) selectingTodo() bool {
	i := ish.cardIndex()
	return i >= 0 && ish.Conversation[i].selectingTodo
}

// updateTodos handles a key press while a card's todo items are being
// selected, reporting whether it was handled. Other keys work as they do in
// the issue view.
func (ish *Issue) updateTodos(msg tea.KeyPressMsg) bool {
	i := ish.cardIndex()
	card := &ish.Conversation[i]

	switch {
	case key.Matches(msg, todoKeys.Up):
		card.moveTodo(-1)

	case key.Matches(msg, todoKeys.Down):
		card.moveTodo(1)

	case key.Matches(msg, todoKeys.Toggle):
		card.toggleTodo()
		ish.dirty = true
		// After a failed save, a new change may fix the problem, so the next
		// exit tries to save again.
		ish.saveErr = nil

	case key.Matches(msg, todoKeys.Done):
		card.stopTodos()

	default:
		return false
	}

	ish.syncViewport()
	ish.revealTodo()

	return true
}

// revealTodo scrolls the viewport so the selected todo item is visible.
func (ish *Issue) revealTodo() {
	i := ish.cardIndex()

	if i < 0 {
		return
	}

	row := ish.Conversation[i].todoRow()

	if row < 0 {
		return
	}

	line := ish.spans[i].top + row
	top, area := ish.viewport.YOffset(), ish.viewport.Height()

	if line < top {
		ish.viewport.SetYOffset(line)
	} else if line >= top+area {
		ish.viewport.SetYOffset(line - area + 1)
	}
}

// editKeyMap returns the key bindings for editing the field under the
// cursor.
func (ish Issue) editKeyMap() editKeyMap {
	return editKeys(ish.cardIndex() >= 0)
}

const (
	ISO8601A  = "2006-01-02 15:04:05Z0700"
	ISO8601B  = "2006-01-02T15:04:05Z0700"
	Timestamp = "2006-01-02 15:04:05 -0700" // Local time :sweat-smile:
	DateTime  = "Jan 2, 2006"               // 15:04:05 MST"
)

// type Cursor struct {
// }

// func (c Cursor) Next() {
// }

func InitialModel() Issue {
	return Issue{
		//
		Id:          "00000",
		DateCreated: "2023-05-26T23:35:36-0400",
		Author:      "brendanberg",
		Title:       "What is ish?",

		Metadata: []Tag{
			{
				Key:    "state",
				Values: []string{"in progress"},
			},
			{
				Key:    "assignees",
				Values: []string{"brendanberg", "jhc"},
			},
		},
		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		//conversation: make([]card, 2),
		Conversation: []Card{
			{
				Id:          "001",
				DateCreated: "2023-05-27T13:41:50-0400",
				Author:      "brendanberg",
				Content:     nil,
			},
			{
				Id:          "002",
				DateCreated: "2023-05-27T23:37:18-0400",
				Author:      "brendanberg",
				Content:     nil,
			},
		},

		selectedComment: 0,
		cursor:          0,
	}
}

func (ish Issue) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (ish Issue) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ish.initComponents()

	if ish.editing {
		if cmd, handled := ish.updateEditing(msg); handled {
			return ish, cmd
		}
	}

	switch msg := msg.(type) {

	case tea.BackgroundColorMsg:
		ish.light = !msg.IsDark()
		ish.help.Styles = help.DefaultStyles(msg.IsDark())

		// An edit may have started before the background color was known.
		if ish.editing {
			ish.styleEditor()
		}

		return ish, nil

	case tea.WindowSizeMsg:
		ish.width = msg.Width
		ish.height = msg.Height
		ish.resize()

		if ish.editing {
			ish.sizeEditor()
			ish.syncViewport()
		}

		if i := ish.cardIndex(); i >= 0 {
			ish.reveal(ish.spans[i], true)
		}

		// A blank issue opens with its title ready to be named.
		if !ish.shown {
			ish.shown = true

			if strings.TrimSpace(ish.Title) == "" && !ish.editing {
				ish.setCursor(1)
				cmd := ish.startEdit()
				return ish, cmd
			}
		}

		return ish, nil

	case tea.KeyPressMsg:
		if ish.confirmingDelete() {
			ish.updateDelete(msg)
			return ish, nil
		}

		if ish.selectingTodo() && ish.updateTodos(msg) {
			return ish, nil
		}

		keys := ish.keyMap()

		switch {
		case key.Matches(msg, keys.Quit):
			cmd := ish.exit(tea.Quit)
			return ish, cmd

		case key.Matches(msg, keys.Back):
			cmd := ish.exit(func() tea.Msg { return CloseMsg{} })
			return ish, cmd

		case key.Matches(msg, keys.Edit):
			cmd := ish.startEdit()
			return ish, cmd

		case key.Matches(msg, keys.Comment):
			cmd := ish.addComment()
			return ish, cmd

		case key.Matches(msg, keys.AddTag):
			cmd := ish.addTag()
			return ish, cmd

		case key.Matches(msg, keys.DeleteTag):
			ish.Metadata[ish.tagIndex()].deleting = true
			return ish, nil

		case key.Matches(msg, keys.Todos):
			ish.Conversation[ish.cardIndex()].startTodos()
			ish.syncViewport()
			ish.revealTodo()
			return ish, nil

		case key.Matches(msg, keys.Help):
			ish.help.ShowAll = !ish.help.ShowAll
			ish.resize()
			return ish, nil

		case key.Matches(msg, keys.Next, keys.Prev):
			total := len(ish.Metadata) + len(ish.Conversation) + 2
			next := key.Matches(msg, keys.Next)
			c := ish.cursor

			if next {
				c = (c + 1) % total
			} else {
				c = (c - 1 + total) % total
			}

			ish.setCursor(c)

			if i := ish.cardIndex(); i >= 0 {
				ish.reveal(ish.spans[i], next)
			}

			return ish, nil

		case key.Matches(msg, keys.Down):
			ish.scrollDown()
			return ish, nil

		case key.Matches(msg, keys.Up):
			ish.scrollUp()
			return ish, nil
		}
	}

	// Paging keys and the mouse wheel scroll the viewport directly.
	var cmd tea.Cmd
	ish.viewport, cmd = ish.viewport.Update(msg)

	return ish, cmd
}

// initComponents sets up the viewport and help footer the first time the
// issue receives a message, since issues are built as struct literals.
func (ish *Issue) initComponents() {
	if ish.ready {
		return
	}

	ish.viewport = viewport.New()
	// Arrow keys move between cards, so the viewport mustn't handle them too.
	ish.viewport.KeyMap.Up.Unbind()
	ish.viewport.KeyMap.Down.Unbind()

	ish.help = help.New()
	ish.ready = true

	ish.syncViewport()
}

// resize fits the viewport into the space left between the pinned header and
// the help footer.
func (ish *Issue) resize() {
	ish.help.SetWidth(ish.width)

	header := lipgloss.Height(ish.renderHeader())
	footer := lipgloss.Height(ish.renderFooter())

	ish.viewport.SetWidth(ish.width)
	ish.viewport.SetHeight(max(ish.height-header-footer, 0))

	ish.syncViewport()
}

// syncViewport re-renders the cards into the viewport, recording where each
// card sits so the selection can be scrolled into view.
func (ish *Issue) syncViewport() {
	content, spans := ish.renderCards()

	ish.spans = spans
	ish.viewport.SetContent(content)
	// Content may have shrunk, so clamp the offset to the new bounds.
	ish.viewport.SetYOffset(ish.viewport.YOffset())
}

// span is the vertical extent, in lines, of a rendered card within the
// viewport content.
type span struct {
	top    int
	height int
}

func (s span) bottom() int {
	return s.top + s.height
}

// firstCard is the cursor position of the first card in the conversation.
// Position 0 selects nothing, position 1 is the title, and the positions
// between the title and the first card are the metadata tags.
func (ish Issue) firstCard() int {
	return len(ish.Metadata) + 2
}

// tagIndex returns the index of the selected metadata tag, or -1 if the
// cursor is not on a tag.
func (ish Issue) tagIndex() int {
	i := ish.cursor - 2
	if i < 0 || i >= len(ish.Metadata) {
		return -1
	}
	return i
}

// updateEditing handles a message while a field is being edited, reporting
// whether it was handled. Keys other than those to save, cancel, or quit go
// to the field's editor.
func (ish *Issue) updateEditing(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg, tea.BackgroundColorMsg:
		return nil, false

	case tea.KeyPressMsg:
		keys := ish.editKeyMap()

		switch {
		case key.Matches(msg, keys.Quit):
			// The field being edited is discarded, but earlier edits are
			// saved.
			ish.stopEdit(false)
			return ish.exit(tea.Quit), true

		case key.Matches(msg, keys.Save):
			if cmd, handled := ish.saveNewTag(); handled {
				return cmd, true
			}

			ish.stopEdit(true)
			return nil, true

		case key.Matches(msg, keys.Cancel):
			ish.stopEdit(false)
			return nil, true
		}
	}

	var cmd tea.Cmd

	if i := ish.tagIndex(); i >= 0 {
		var tag tea.Model
		tag, cmd = ish.Metadata[i].Update(msg)
		ish.Metadata[i] = tag.(Tag)
	} else if i := ish.cardIndex(); i >= 0 {
		var card tea.Model
		card, cmd = ish.Conversation[i].Update(msg)
		ish.Conversation[i] = card.(Card)

		// The editor grows and shrinks with its content.
		ish.syncViewport()
		ish.reveal(ish.spans[i], true)
	} else {
		ish.titleInput, cmd = ish.titleInput.Update(msg)
	}

	return cmd, true
}

// startEdit begins editing the field under the cursor: the title text, the
// values of a metadata tag, or the markdown content of a card.
func (ish *Issue) startEdit() tea.Cmd {
	var cmd tea.Cmd
	dark := !ish.light

	switch {
	case ish.cursor == 1:
		ish.titleInput = textinput.New()
		ish.titleInput.Prompt = ""
		ish.titleInput.SetStyles(textinput.DefaultStyles(dark))
		ish.titleInput.SetValue(ish.Title)
		ish.titleInput.CursorEnd()
		cmd = ish.titleInput.Focus()

	case ish.tagIndex() >= 0:
		cmd = ish.Metadata[ish.tagIndex()].startEdit(ish.headerWidth(), dark)

	case ish.cardIndex() >= 0:
		width, height := ish.cardArea()
		cmd = ish.Conversation[ish.cardIndex()].startEdit(width, height, dark)

	default:
		return nil
	}

	ish.editing = true
	// After a failed save, a new edit may fix the problem, so the next exit
	// tries to save again.
	ish.saveErr = nil
	ish.sizeEditor()
	// The help footer changes while editing.
	ish.resize()

	if i := ish.cardIndex(); i >= 0 {
		ish.reveal(ish.spans[i], true)
	}

	return cmd
}

// stopEdit stops editing the field under the cursor, keeping the edited value
// if save is true and marking the issue as having unsaved changes if the
// value changed. A title left blank is not kept.
func (ish *Issue) stopEdit(save bool) {
	switch {
	case ish.cursor == 1:
		title := strings.TrimSpace(ish.titleInput.Value())

		if save && title != "" && title != ish.Title {
			ish.Title = title
			ish.dirty = true
		}

	case ish.tagIndex() >= 0:
		switch {
		// A new tag that's cancelled is removed, since it's missing its key
		// or its values, and can't be written to the issue file without
		// both.
		case ish.newTag && !save:
			ish.removeTag(ish.tagIndex())
		case save:
			ish.dirty = ish.Metadata[ish.tagIndex()].commitEdit() || ish.dirty
		default:
			ish.Metadata[ish.tagIndex()].cancelEdit()
		}

	case ish.cardIndex() >= 0:
		card := &ish.Conversation[ish.cardIndex()]

		switch {
		// A new comment that's cancelled or left empty is removed.
		case ish.newComment && (!save || strings.TrimSpace(card.editor.Value()) == ""):
			card.cancelEdit()
			ish.removeNewComment()
		case save:
			ish.dirty = card.commitEdit() || ish.dirty
		default:
			card.cancelEdit()
		}
	}

	ish.editing = false
	ish.newComment = false
	ish.newTag = false
	ish.resize()

	if i := ish.cardIndex(); i >= 0 {
		ish.reveal(ish.spans[i], true)
	}
}

// styleEditor styles the editor of the field under the cursor for the
// terminal's background.
func (ish *Issue) styleEditor() {
	dark := !ish.light

	switch {
	case ish.cursor == 1:
		ish.titleInput.SetStyles(textinput.DefaultStyles(dark))

	case ish.tagIndex() >= 0:
		ish.Metadata[ish.tagIndex()].input.SetStyles(textinput.DefaultStyles(dark))
		ish.Metadata[ish.tagIndex()].keyInput.SetStyles(textinput.DefaultStyles(dark))

	case ish.cardIndex() >= 0:
		ish.Conversation[ish.cardIndex()].editor.SetStyles(textarea.DefaultStyles(dark))
	}
}

// sizeEditor fits the editor of the field under the cursor to the screen.
func (ish *Issue) sizeEditor() {
	switch {
	case ish.cursor == 1:
		ish.titleInput.SetWidth(max(ish.headerWidth()-lipgloss.Width(ish.titleLabel()), 1))

	case ish.tagIndex() >= 0:
		ish.Metadata[ish.tagIndex()].sizeEditor(ish.headerWidth())

	case ish.cardIndex() >= 0:
		width, height := ish.cardArea()
		ish.Conversation[ish.cardIndex()].sizeEditor(width, height)
	}
}

// headerWidth returns the width available to the title and metadata.
func (ish Issue) headerWidth() int {
	return ish.width - styles.Header.GetHorizontalFrameSize()
}

// cardArea returns the size available to a card in the viewport.
func (ish Issue) cardArea() (int, int) {
	return ish.viewport.Width() - styles.Body.GetHorizontalFrameSize(), ish.viewport.Height()
}

// titleLabel renders the issue ID shown before the title while it's being
// edited. The ID is never editable.
func (ish Issue) titleLabel() string {
	return styles.TitleSelected.Render(fmt.Sprintf("[%s]", ish.Id)) + " "
}

// cardIndex returns the index of the selected card, or -1 if the cursor is
// not on a card.
func (ish Issue) cardIndex() int {
	i := ish.cursor - ish.firstCard()
	if i < 0 || i >= len(ish.Conversation) {
		return -1
	}
	return i
}

// setCursor moves the cursor to c, updating the selection state of the
// metadata tag or card under the old and new positions.
func (ish *Issue) setCursor(c int) {
	ish.selectAt(ish.cursor, false)
	ish.cursor = c
	ish.selectAt(ish.cursor, true)

	ish.syncViewport()
}

func (ish *Issue) selectAt(c int, selected bool) {
	first := ish.firstCard()

	switch {
	case c >= 2 && c < first:
		meta, _ := ish.Metadata[c-2].Update(SelectMsg(selected))
		ish.Metadata[c-2] = meta.(Tag)
	case c >= first && c < first+len(ish.Conversation):
		comment, _ := ish.Conversation[c-first].Update(SelectMsg(selected))
		ish.Conversation[c-first] = comment.(Card)
	}
}

// scrollDown scrolls the rest of the selected card into view one line at a
// time, then moves the selection to the next card once the selected card's
// last line is visible.
func (ish *Issue) scrollDown() {
	i := ish.cardIndex()
	bottom := ish.viewport.YOffset() + ish.viewport.Height()

	switch {
	case len(ish.spans) == 0:
		return
	case i < 0:
		ish.setCursor(ish.firstCard())
		ish.reveal(ish.spans[0], true)
	case ish.spans[i].bottom() > bottom:
		ish.viewport.ScrollDown(1)
	case i+1 < len(ish.spans):
		ish.setCursor(ish.cursor + 1)
		ish.reveal(ish.spans[i+1], true)
	}
}

// scrollUp scrolls the rest of the selected card into view one line at a
// time, then moves the selection to the previous card once the selected
// card's first line is visible.
func (ish *Issue) scrollUp() {
	i := ish.cardIndex()
	last := len(ish.spans) - 1

	switch {
	case len(ish.spans) == 0:
		return
	case i < 0:
		ish.setCursor(ish.firstCard() + last)
		ish.reveal(ish.spans[last], false)
	case ish.spans[i].top < ish.viewport.YOffset():
		ish.viewport.ScrollUp(1)
	case i > 0:
		ish.setCursor(ish.cursor - 1)
		ish.reveal(ish.spans[i-1], false)
	}
}

// reveal scrolls the viewport so that s is visible. A card taller than the
// viewport is revealed from its top edge when moving down and from its bottom
// edge when moving up.
func (ish *Issue) reveal(s span, down bool) {
	top := ish.viewport.YOffset()
	area := ish.viewport.Height()

	if s.height > area {
		if down && (s.top < top || s.top >= top+area) {
			ish.viewport.SetYOffset(s.top)
		} else if !down && (s.bottom() <= top || s.bottom() > top+area) {
			ish.viewport.SetYOffset(s.bottom() - area)
		}
		return
	}

	if s.top < top {
		ish.viewport.SetYOffset(s.top)
	} else if s.bottom() > top+area {
		ish.viewport.SetYOffset(s.bottom() - area)
	}
}

// renderHeader renders the title, byline, metadata tags, and the rule beneath
// them, all of which stay pinned to the top of the screen.
func (ish Issue) renderHeader() string {
	t, err := ish.Opened()

	if err != nil {
		log.Fatal(err)
	}

	c := make([]string, 0)

	if ish.editing && ish.cursor == 1 {
		c = append(c, ish.titleLabel()+ish.titleInput.View())
	} else {
		c = append(c, ish.getTitleStyle().Render(fmt.Sprintf("[%s] %s", ish.Id, ish.Title)))
	}
	c = append(c, styles.BylineBlock.Copy().MarginBottom(1).Render(
		styles.BylineEmphasis.Render(ish.Author)+
			styles.Byline.Render(" opened this issue on "+t.Format(DateTime)),
	))

	for _, tag := range ish.Metadata {
		c = append(c, tag.View().Content)
	}

	header := styles.Header.Render(lipgloss.JoinVertical(lipgloss.Left, c...))

	// Separate the header from the cards with a rule spanning the screen, or
	// the header itself before the terminal size is known.
	width := ish.width
	if width <= 0 {
		width = lipgloss.Width(header)
	}

	rule := styles.Rule.Render(
		strings.Repeat("─", max(width-styles.Rule.GetHorizontalMargins(), 0)),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, rule)
}

// renderCards renders every card in the conversation, along with the position
// of each card in the result.
func (ish Issue) renderCards() (string, []span) {
	c := make([]string, 0, len(ish.Conversation))
	spans := make([]span, 0, len(ish.Conversation))
	top := 0

	for _, card := range ish.Conversation {
		s := card.View().Content
		h := lipgloss.Height(s)

		c = append(c, s)
		spans = append(spans, span{top: top, height: h})
		top += h
	}

	return styles.Body.Render(lipgloss.JoinVertical(lipgloss.Left, c...)), spans
}

// renderFooter renders the key binding help shown below the viewport.
func (ish Issue) renderFooter() string {
	help := ish.help.View(ish.keyMap())

	switch {
	case ish.editing:
		help = ish.help.View(ish.editKeyMap())
	case ish.confirmingDelete():
		help = ish.help.View(deleteKeys)
	case ish.selectingTodo():
		help = ish.help.View(todoKeys)
	}

	if ish.saveErr == nil {
		return styles.Footer.Render(help)
	}

	message := styles.Footer
	if ish.width > 0 {
		message = message.Width(ish.width)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		message.Render(styles.Error.Render(fmt.Sprintf(
			"Couldn't save changes: %v. Leave again to discard them.", ish.saveErr,
		))),
		styles.Footer.Render(help),
	)
}

func (ish Issue) View() tea.View {
	content := lipgloss.JoinVertical(lipgloss.Left,
		ish.renderHeader(),
		ish.viewport.View(),
		ish.renderFooter(),
	)

	if ish.width > 0 && ish.height > 0 {
		content = lipgloss.NewStyle().
			MaxWidth(ish.width).
			MaxHeight(ish.height).
			Render(content)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	return v
}

// Opened returns the time the issue was opened, from its byline.
func (ish Issue) Opened() (time.Time, error) {
	return ParseDate(ish.DateCreated)
}

// dateLayouts are the formats bylines' dates are written in.
var dateLayouts = []string{time.RFC3339, ISO8601A, ISO8601B, Timestamp}

// ParseDate parses a byline's date, which may be in any of dateLayouts.
func ParseDate(s string) (time.Time, error) {
	var err error

	for _, layout := range dateLayouts {
		var t time.Time

		if t, err = time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, err
}

// State returns the values of the issue's state tag, or an empty string if
// it has none.
func (ish Issue) State() string {
	return ish.tagValues("state")
}

// Assignees returns the values of the issue's assignees tag, or an empty
// string if it has none.
func (ish Issue) Assignees() string {
	return ish.tagValues("assignees")
}

// tagValues returns the values of the issue's tags with any of the given
// keys, separated by commas.
func (ish Issue) tagValues(keys ...string) string {
	var values []string

	for _, tag := range ish.Metadata {
		if slices.Contains(keys, tag.Key) {
			values = append(values, tag.Values...)
		}
	}

	return strings.Join(values, ", ")
}

func (ish Issue) Print() {
	fmt.Printf("Title: %s\n", ish.Title)
	fmt.Printf("Author: %s\n", ish.Author)
	fmt.Printf("Date: %s\n", ish.DateCreated)

	if len(ish.Metadata) > 0 {
		fmt.Print("Metadata:\n")
		for _, elt := range ish.Metadata {
			fmt.Printf("  Key('%s'):", elt.Key)
			for _, val := range elt.Values {
				fmt.Printf(" Value('%s')", val)
			}
			fmt.Print("\n")
		}
	}

	for _, card := range ish.Conversation {
		fmt.Print("-----\n")
		fmt.Printf("Author: %s\n", card.Author)
		fmt.Printf("Date: %s\n", card.DateCreated)
		fmt.Print("Content:\n  ")
		fmt.Print(strings.ReplaceAll(*card.Content, "\n", "\n  "))
	}

}

func (ish Issue) getTitleStyle() lipgloss.Style {
	switch ish.cursor {
	case 1:
		return styles.TitleSelected
	default:
		return styles.Title
	}
}

// NewIssue returns an empty issue with the given ID, created by author at
// created, with one empty comment by the same author.
func NewIssue(id, author string, created time.Time) Issue {
	date := created.Format(time.RFC3339)
	content := ""

	return Issue{
		Id:          id,
		DateCreated: date,
		Author:      author,
		Metadata:    []Tag{},
		Conversation: []Card{
			{
				DateCreated: date,
				Author:      author,
				Content:     &content,
			},
		},
	}
}
