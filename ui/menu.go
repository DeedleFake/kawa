package ui

import (
	"image"

	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

type Menu struct {
	items    []*MenuItem
	itemInfo map[*MenuItem]image.Rectangle
	bounds   image.Rectangle
}

func NewMenu(items ...*MenuItem) *Menu {
	m := Menu{
		items:    make([]*MenuItem, 0, len(items)),
		itemInfo: make(map[*MenuItem]image.Rectangle, len(items)),
	}
	for _, item := range items {
		m.Add(item)
	}
	return &m
}

func (m *Menu) Len() int {
	return len(m.items)
}

func (m *Menu) updateBounds() {
	r := image.ZR
	for _, ib := range m.itemInfo {
		r = r.Union(ib)
	}
	m.bounds = r
}

func (m *Menu) Bounds() (b image.Rectangle) {
	return m.bounds
}

func (m *Menu) ItemAt(p image.Point) (*MenuItem, bool) {
	for item, ib := range m.itemInfo {
		if p.In(ib) {
			return item, true
		}
	}
	return nil, false
}

func (m *Menu) ItemBounds(item *MenuItem) (image.Rectangle, bool) {
	b, ok := m.itemInfo[item]
	return b, ok
}

func (m *Menu) Click(p image.Point) {
	item, ok := m.ItemAt(p)
	if ok {
		item.OnSelect()
	}
}

func (m *Menu) Add(item *MenuItem) {
	m.items = append(m.items, item)
	b := image.Rect(0, 0, item.active.Width(), item.active.Height()).Add(image.Pt(
		(m.bounds.Dx()/2)-(item.active.Width()/2),
		m.bounds.Max.Y+WindowBorder,
	)).Inset(-WindowBorder)
	m.itemInfo[item] = b
	m.bounds = m.bounds.Union(b)
}

func (m *Menu) Remove(i int) {
	item := m.items[i]
	m.items = slices.Delete(m.items, i, i+1)
	delete(m.itemInfo, item)
	m.updateBounds()
}

type MenuItem struct {
	OnSelect func()

	active   wlr.Texture
	inactive wlr.Texture
}

func NewMenuItem(active, inactive wlr.Texture) *MenuItem {
	if (active.Width() != inactive.Width()) || (active.Height() != inactive.Height()) {
		panic("active and inactive sizes do no match")
	}

	return &MenuItem{
		active:   active,
		inactive: inactive,
	}
}

func (item *MenuItem) Destroy() {
	item.active.Destroy()
	item.inactive.Destroy()
}
