package ui

import (
	"image"

	"deedles.dev/kawa/theme"
	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type Menu struct {
	items []*MenuItem
	prev  *MenuItem
}

func NewMenu(items ...*MenuItem) *Menu {
	return &Menu{
		items: items,
	}
}

func (m *Menu) Len() int {
	return len(m.items)
}

func (m *Menu) forEach(cb func(*MenuItem, image.Rectangle) bool) {
	var p image.Point
	for _, t := range m.items {
		r := image.Rect(0, 0, t.active.Width(), t.active.Height()).Add(p)
		if !cb(t, r) {
			return
		}
		p = r.Max
	}
}

func (m *Menu) Bounds() (b image.Rectangle) {
	m.forEach(func(item *MenuItem, r image.Rectangle) bool {
		b = b.Union(r)
		return true
	})
	return b.Inset(-theme.WindowBorder).Add(image.Pt(theme.WindowBorder, theme.WindowBorder))
}

func (m *Menu) StartOffset() (p image.Point) {
	m.forEach(func(item *MenuItem, r image.Rectangle) bool {
		if item != m.prev {
			return true
		}
		p = r.Min.Add(r.Max).Div(2)
		return false
	})
	return p
}

func (m *Menu) Click(p image.Point) {
	m.forEach(func(item *MenuItem, r image.Rectangle) bool {
		if !p.In(r) {
			return true
		}
		if cb := item.OnSelect; cb != nil {
			cb()
		}
		m.prev = item
		return false
	})
}

func (m *Menu) Add(item *MenuItem) {
	m.items = append(m.items, item)
}

func (m *Menu) Remove(i int) {
	m.items[i].Destroy()
	m.items = slices.Delete(m.items, i, i+1)
}

func CreateTextTexture(renderer wlr.Renderer, src image.Image, face font.Face, item string) wlr.Texture {
	fdraw := font.Drawer{
		Src:  src,
		Face: face,
		Dot:  fixed.P(0, int(fontOptions.Size)),
	}

	extents, _ := fdraw.BoundString(item)
	buf := image.NewNRGBA(image.Rect(
		0,
		0,
		(extents.Max.X - extents.Min.X).Floor(),
		int(fontOptions.Size),
	))
	fdraw.Dst = buf
	fdraw.DrawString(item)

	return wlr.TextureFromImage(renderer, buf)
}

type MenuItem struct {
	OnSelect func()

	active   wlr.Texture
	inactive wlr.Texture
}

func (item *MenuItem) Destroy() {
	item.active.Destroy()
	item.inactive.Destroy()
}
