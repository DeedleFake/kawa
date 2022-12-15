package main

import (
	"image"

	"deedles.dev/kawa/draw"
	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
	"golang.org/x/exp/slices"
)

var (
	menuItemInset = geom.Pt(WindowBorder, WindowBorder)
)

type Menu struct {
	items  []*MenuItem
	bounds []geom.Rect[float64]
	prev   *MenuItem
}

func NewMenu(items ...*MenuItem) *Menu {
	m := Menu{
		items:  make([]*MenuItem, 0, len(items)),
		bounds: make([]geom.Rect[float64], 0, len(items)),
	}
	for _, item := range items {
		m.add(item)
	}
	m.updateBounds()
	return &m
}

func (m *Menu) Len() int {
	return len(m.items)
}

func (m *Menu) updateBounds() {
	geom.ArrangeVerticalStack(m.bounds)
}

func (m *Menu) Item(i int) *MenuItem {
	return m.items[i]
}

func (m *Menu) Bounds() (b geom.Rect[float64]) {
	return geom.Rect[float64]{
		Min: m.bounds[0].Min,
		Max: m.bounds[len(m.bounds)-1].Max,
	}
}

func (m *Menu) Select(item *MenuItem) {
	i := slices.Index(m.items, item)
	if i < 0 {
		return
	}

	item.OnSelect()
	m.prev = item
}

func (m *Menu) Prev() *MenuItem {
	i := slices.Index(m.items, m.prev)
	if i < 0 {
		return nil
	}
	return m.prev
}

func (m *Menu) ItemAt(p geom.Point[float64]) *MenuItem {
	for i, ib := range m.bounds {
		if p.In(ib) {
			return m.items[i]
		}
	}
	return nil
}

func (m *Menu) ItemBounds(item *MenuItem) geom.Rect[float64] {
	i := slices.Index(m.items, item)
	if i < 0 {
		return geom.Rect[float64]{}
	}
	return m.bounds[i]
}

func (m *Menu) add(item *MenuItem) {
	m.items = append(m.items, item)
	m.bounds = append(m.bounds, geom.Rect[float64]{
		Max: geom.PConv[float64](item.Size().Add(menuItemInset)),
	})
}

func (m *Menu) Add(item *MenuItem) {
	m.add(item)
	m.updateBounds()
}

func (m *Menu) Remove(item *MenuItem) {
	i := slices.Index(m.items, item)
	m.items = slices.Delete(m.items, i, i+1)
	m.bounds = slices.Delete(m.bounds, i, i+1)
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

func NewTextMenuItem(renderer wlr.Renderer, text string) *MenuItem {
	return NewMenuItem(
		draw.CreateTextTexture(renderer, image.White, text),
		draw.CreateTextTexture(renderer, image.Black, text),
	)
}

func (item *MenuItem) Size() geom.Point[int] {
	return geom.Rt(0, 0, item.active.Width(), item.active.Height()).
		Union(geom.Rt(0, 0, item.inactive.Width(), item.inactive.Height())).
		Size()
}

func (item *MenuItem) Release() {
	item.active.Destroy()
	item.inactive.Destroy()
}
