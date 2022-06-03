package main

import (
	"deedles.dev/kawa/geom"
	"deedles.dev/kawa/internal/util"
	"golang.org/x/exp/slices"
)

type Container struct {
	children []Widget
	sizes    []geom.Rect[float64]
	bounds   []geom.Rect[float64]
	layout   ContainerLayout
}

func NewContainer(layout ContainerLayout, children ...Widget) *Container {
	return &Container{
		children: children,
		layout:   layout,
	}
}

func (c *Container) Add(child Widget) {
	c.children = append(c.children, child)
}

func (c *Container) Remove(child Widget) {
	i := slices.IndexFunc(c.children, util.Match(child))
	c.RemoveIndex(i)
}

func (c *Container) RemoveIndex(i int) {
	c.children = slices.Delete(c.children, i, i+1)
}

func (c *Container) Children() []Widget {
	return c.children
}

func (c *Container) Size(min, max geom.Point[float64]) geom.Point[float64] {
	c.sizes = c.layout.LayoutChildren(c, min, max)

	var bounds geom.Rect[float64]
	for _, b := range c.sizes {
		bounds = bounds.Union(b)
	}
	return bounds.Size()
}

func (c *Container) Position(base geom.Rect[float64]) (bounds geom.Rect[float64]) {
	c.bounds = c.bounds[:0]
	for i, child := range c.Children() {
		cb := child.Position(c.layout.Position(base, c.sizes[i]))
		c.bounds = append(c.bounds, cb)
		bounds = bounds.Union(cb)
	}

	return bounds
}

func (c *Container) Render(server *Server, out *Output) {
	for _, c := range c.children {
		c.Render(server, out)
	}
}

type ContainerLayout interface {
	LayoutChildren(c *Container, min, max geom.Point[float64]) []geom.Rect[float64]
	Position(base, layout geom.Rect[float64]) geom.Rect[float64]
}

var (
	VBoxLayout  ContainerLayout = boxLayout{vert: true}
	HBoxLayout  ContainerLayout = boxLayout{vert: false}
	StackLayout ContainerLayout = stackLayout{}
)

type boxLayout struct {
	vert bool
}

func (b boxLayout) addOffset(base geom.Rect[float64], off geom.Point[float64]) geom.Rect[float64] {
	if b.vert {
		return base.Add(geom.Pt(0, off.Y))
	}
	return base.Add(geom.Pt(off.X, 0))
}

func (b boxLayout) childMax(base, layout geom.Rect[float64]) geom.Point[float64] {
	if b.vert {
		return geom.Pt(base.Max.X, layout.Max.Y)
	}
	return geom.Pt(layout.Max.X, base.Max.Y)
}

func (b boxLayout) LayoutChildren(c *Container, min, max geom.Point[float64]) (sizes []geom.Rect[float64]) {
	var bounds geom.Rect[float64]
	for _, c := range c.Children() {
		off := b.addOffset(geom.Rect[float64]{}, bounds.Max)

		cmax := max.Sub(off.Min)
		csize := c.Size(min, cmax)
		sizes = append(sizes, off.Resize(csize))
	}

	return sizes
}

func (b boxLayout) Position(base, layout geom.Rect[float64]) geom.Rect[float64] {
	return geom.Rect[float64]{Min: base.Min.Add(layout.Min), Max: b.childMax(base, layout)}
}

type stackLayout struct{}

func (s stackLayout) LayoutChildren(c *Container, min, max geom.Point[float64]) (sizes []geom.Rect[float64]) {
	for _, c := range c.Children() {
		sizes = append(sizes, geom.Rect[float64]{Max: c.Size(min, max)})
	}
	return sizes
}

func (s stackLayout) Position(base, layout geom.Rect[float64]) geom.Rect[float64] {
	return geom.Rect[float64]{Min: base.Min, Max: layout.Max}
}
