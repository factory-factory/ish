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
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Next         key.Binding
	Prev         key.Binding
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
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "previous card"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "next card"),
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
	return []key.Binding{k.Up, k.Down, k.Next, k.Back, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Next, k.Prev},
		{k.PageUp, k.PageDown, k.HalfPageUp, k.HalfPageDown},
		{k.Back, k.Help, k.Quit},
	}
}
