package models

import (
	"cmp"
	"slices"
)

// item is a selectable item in a card's content: a todo item, or a link to
// another issue.
type item struct {
	// offset is where the item starts in the content, in bytes, which puts
	// items in reading order.
	offset int

	// todo is the index of the todo item's line, or -1 for a link.
	todo int

	// link is the link's reference, for a link.
	link reference
}

// items returns the card's todo items and links, in reading order.
func (c Card) items() []item {
	if c.Content == nil {
		return nil
	}

	var items []item

	offsets := todoOffsets(*c.Content)

	for i, line := range todoLines(*c.Content) {
		items = append(items, item{offset: offsets[i], todo: line})
	}

	for _, ref := range contentReferences(*c.Content, c.issues) {
		items = append(items, item{offset: ref.start, todo: -1, link: ref})
	}

	// A todo item comes before a link at the start of its text.
	slices.SortStableFunc(items, func(a, b item) int { return cmp.Compare(a.offset, b.offset) })

	return items
}

// startItems begins selecting the card's items, starting with the first,
// reporting whether it has any.
func (c *Card) startItems() bool {
	if len(c.items()) == 0 {
		return false
	}

	c.selecting = true
	c.item = 0

	return true
}

// stopItems stops selecting the card's items.
func (c *Card) stopItems() {
	c.selecting = false
}

// moveItem moves the selection by delta items, stopping at the first and
// last items.
func (c *Card) moveItem(delta int) {
	c.item = min(max(c.item+delta, 0), len(c.items())-1)
}

// selectedItem returns the selected item, if items are being selected.
func (c Card) selectedItem() (item, bool) {
	items := c.items()

	if !c.selecting || c.item < 0 || c.item >= len(items) {
		return item{}, false
	}

	return items[c.item], true
}

// markItems returns the card's content with markers inserted to find its
// links and the selected item once it's rendered.
func (c Card) markItems() string {
	if c.Content == nil {
		return ""
	}

	// An insertion puts text at offset, replacing the remove bytes there.
	type insertion struct {
		offset int
		text   string
		remove int
	}

	var insertions []insertion
	selected, selecting := c.selectedItem()

	if selecting && selected.todo >= 0 {
		insertions = append(insertions, insertion{selected.offset, todoMarker, 0})
	}

	for _, it := range c.items() {
		if it.todo >= 0 {
			continue
		}

		marker := linkMarker
		if selecting && selected.todo < 0 && it.offset == selected.offset {
			marker = selectedLinkMarker
		}

		insertions = append(insertions,
			insertion{it.link.start, marker, 0},
			insertion{it.link.end, linkEndMarker, 0},
		)
	}

	// Users' emails are shown as their names.
	for _, m := range c.dir.mentions(*c.Content) {
		insertions = append(insertions, insertion{m.start, mentionMarker + mentionPrefix + m.name + mentionEndMarker, m.end - m.start})
	}

	// Insert from the end, so earlier offsets stay valid. At the same
	// offset, the insertion listed first ends up first.
	slices.SortStableFunc(insertions, func(a, b insertion) int { return cmp.Compare(a.offset, b.offset) })
	content := *c.Content

	for i := len(insertions) - 1; i >= 0; i-- {
		in := insertions[i]
		content = content[:in.offset] + in.text + content[in.offset+in.remove:]
	}

	return content
}
