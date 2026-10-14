package dropdownv2

import tea "charm.land/bubbletea/v2"

func keyIsEnterOrSpace(msg tea.KeyPressMsg) bool {
	k := msg.Key()
	if k.Code == tea.KeyEnter || k.Code == tea.KeyReturn || k.Code == tea.KeySpace {
		return true
	}
	s := msg.String()
	return s == "enter" || s == " "
}

func keyIsUp(msg tea.KeyPressMsg) bool {
	k := msg.Key()
	if k.Code == tea.KeyUp {
		return true
	}
	s := msg.String()
	return s == "up" || s == "k"
}

func keyIsDown(msg tea.KeyPressMsg) bool {
	k := msg.Key()
	if k.Code == tea.KeyDown {
		return true
	}
	s := msg.String()
	return s == "down" || s == "j"
}

func keyIsEscape(msg tea.KeyPressMsg) bool {
	k := msg.Key()
	if k.Code == tea.KeyEscape || k.Code == tea.KeyEsc {
		return true
	}
	return msg.String() == "esc"
}
