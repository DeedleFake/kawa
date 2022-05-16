package main

import (
	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type Menu struct {
	items    []*MenuItem
	itemInfo map[*MenuItem]geom.Rect[float64]
	bounds   geom.Rect[float64]
}

func NewMenu(items ...*MenuItem) *Menu {
	m := Menu{
		items:    make([]*MenuItem, 0, len(items)),
		itemInfo: make(map[*MenuItem]geom.Rect[float64], len(items)),
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
	maps.Clear(m.itemInfo)

	bounds := make([]geom.Rect[float64], 0, len(m.items))
	r := geom.Rect[float64]{}
	for _, item := range m.items {
		tb := geom.Rt(0, 0, float64(item.active.Width())+2*WindowBorder, float64(item.active.Height())+2*WindowBorder)
		if tb.Dx() < r.Dx() {
			tb.Max.X = r.Max.X
		}
		tb = tb.Add(geom.Pt(0, r.Max.Y))
		bounds = append(bounds, tb)
		r = r.Union(tb)
	}
	m.bounds = r

	for i, b := range bounds {
		item := m.items[i]
		m.itemInfo[item] = geom.Rt(0, 0, r.Dx(), b.Dy()).Add(b.Min)
	}
}

func (m *Menu) Item(i int) *MenuItem {
	return m.items[i]
}

func (m *Menu) Bounds() (b geom.Rect[float64]) {
	return m.bounds
}

func (m *Menu) ItemAt(p geom.Point[float64]) *MenuItem {
	for item, ib := range m.itemInfo {
		if p.In(ib) {
			return item
		}
	}
	return nil
}

func (m *Menu) ItemBounds(item *MenuItem) geom.Rect[float64] {
	return m.itemInfo[item]
}

func (m *Menu) Add(item *MenuItem) {
	m.items = append(m.items, item)
	m.updateBounds()
}

func (m *Menu) Remove(i int) {
	m.items = slices.Delete(m.items, i, i+1)
	m.updateBounds()
}

func (m *Menu) RemoveItem(item *MenuItem) {
	i := slices.Index(m.items, item)
	m.Remove(i)
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
