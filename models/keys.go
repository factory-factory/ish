package models

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
)

// keyMap holds the key bindings for the issue view. It satisfies
// help.KeyMap so the bindings can be listed in the help footer.
type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	NextComment  key.Binding
	PrevComment  key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Next         key.Binding
	Prev         key.Binding
	Edit         key.Binding
	Comment      key.Binding
	AddTag       key.Binding
	DeleteTag    key.Binding
	Select       key.Binding
	Back         key.Binding
	Help         key.Binding
	Quit         key.Binding
}

// keys are the issue view's key bindings. Paging bindings are shared with the
// viewport, which handles them itself.
var keys = func() keyMap {
	pager := viewport.DefaultKeyMap()

	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "scroll down"),
		),
		NextComment: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "next comment"),
		),
		PrevComment: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "previous comment"),
		),
		PageUp:       pager.PageUp,
		PageDown:     pager.PageDown,
		HalfPageUp:   pager.HalfPageUp,
		HalfPageDown: pager.HalfPageDown,
		Next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous field"),
		),
		// Edit is enabled only while the cursor is on an editable field.
		Edit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit"),
			key.WithDisabled(),
		),
		// Comment is enabled only for issues with an author for new comments.
		// See Issue.SetAuthor.
		Comment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comment"),
			key.WithDisabled(),
		),
		AddTag: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "add field"),
		),
		// DeleteTag is enabled only while the cursor is on a metadata tag.
		// Mac keyboards' delete key sends backspace.
		DeleteTag: key.NewBinding(
			key.WithKeys("backspace", "delete"),
			key.WithHelp("⌫", "delete field"),
			key.WithDisabled(),
		),
		// Select is enabled only while the cursor is on a card with todo
		// items or links, or a tag with links.
		Select: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "select"),
			key.WithDisabled(),
		),
		// Back is enabled only for issues that can be closed. See
		// Issue.SetClosable.
		Back: key.NewBinding(
			key.WithKeys("esc", "w"),
			key.WithHelp("esc/w", "back"),
			key.WithDisabled(),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more keys"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}()

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextComment, k.PrevComment, k.Next, k.Edit, k.Comment, k.AddTag, k.DeleteTag, k.Select, k.Back, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextComment, k.PrevComment, k.Next, k.Prev, k.Edit, k.Comment, k.AddTag, k.DeleteTag, k.Select},
		{k.Up, k.Down, k.PageUp, k.PageDown, k.HalfPageUp, k.HalfPageDown},
		{k.Back, k.Help, k.Quit},
	}
}

// editKeyMap holds the key bindings used while a field is being edited. All
// other keys are handled by the field's editor.
type editKeyMap struct {
	Save    key.Binding
	Next    key.Binding
	Suggest key.Binding
	Cancel  key.Binding
	Quit    key.Binding
}

// editKeys returns the key bindings for editing a field. Multiline fields
// use enter for new lines, so they're saved with ctrl+s instead.
func editKeys(multiline bool) editKeyMap {
	save := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "save"),
	)

	if multiline {
		save = key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		)
	}

	return editKeyMap{
		Save: save,
		// Next is enabled only while a new tag's key is being edited, to
		// move on to its values.
		Next: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "values"),
			key.WithDisabled(),
		),
		// Suggest is only listed in the help footer, while there's more
		// than one suggestion to cycle through. The tag's input handles
		// the keys.
		Suggest: key.NewBinding(
			key.WithKeys("tab", "shift+tab"),
			key.WithHelp("tab/shift+tab", "suggestions"),
			key.WithDisabled(),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

func (k editKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Save, k.Next, k.Suggest, k.Cancel}
}

func (k editKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Save, k.Next, k.Suggest, k.Cancel, k.Quit}}
}

// itemKeyMap holds the key bindings used while selecting a card's todo items
// and links, or a tag's links. Other keys work as they do in the issue view.
type itemKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Choose key.Binding
	Done   key.Binding
	Quit   key.Binding
}

var itemKeys = itemKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "previous item"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next item"),
	),
	Choose: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "check/open"),
	),
	Done: key.NewBinding(
		key.WithKeys("left", "h", "esc"),
		key.WithHelp("←/h/esc", "done"),
	),
	Quit: keys.Quit,
}

func (k itemKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Choose, k.Done, k.Quit}
}

func (k itemKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

// deleteKeyMap holds the key bindings used while confirming that a metadata
// tag should be deleted. Any other key cancels.
type deleteKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

var deleteKeys = deleteKeyMap{
	Confirm: key.NewBinding(
		key.WithKeys("backspace", "delete"),
		key.WithHelp("⌫", "delete field"),
	),
	// Cancel is only shown in the help footer; it matches no keys itself.
	Cancel: key.NewBinding(
		key.WithKeys("any"),
		key.WithHelp("any other key", "cancel"),
	),
}

func (k deleteKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Cancel}
}

func (k deleteKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
