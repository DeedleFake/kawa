package geom

import (
	"image"
	"image/color"

	"golang.org/x/exp/constraints"
)

// A Rect contains the points with Min.X <= X < Max.X, Min.Y <= Y < Max.Y. It
// is well-formed if Min.X <= Max.X and likewise for Y. Points are always
// well-formed. A rectangle's methods always return well-formed outputs for
// well-formed inputs.
//
// A Rect is also an Image whose bounds are the rectangle itself. At returns
// color.Opaque for points in the rectangle and color.Transparent otherwise.
type Rect[T constraints.Integer | constraints.Float] struct {
	Min, Max Point[T]
}

// Rt is shorthand for Rect{Pt(x0, y0), Pt(x1, y1)}. The returned
// rectangle has minimum and maximum coordinates swapped if necessary
// so that it is well-formed.
func Rt[T constraints.Integer | constraints.Float](x0, y0, x1, y1 T) Rect[T] {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return Rect[T]{Point[T]{x0, y0}, Point[T]{x1, y1}}
}

// RConv converts a Rect[In] to a Rect[Out] with possible loss of precision.
func RConv[Out constraints.Integer | constraints.Float, In constraints.Integer | constraints.Float](r Rect[In]) Rect[Out] {
	return Rect[Out]{
		Min: PConv[Out](r.Min),
		Max: PConv[Out](r.Max),
	}
}

func (r Rect[T]) Dx() T {
	return r.Max.X - r.Min.X
}

func (r Rect[T]) Dy() T {
	return r.Max.Y - r.Min.Y
}

func (r Rect[T]) Size() Point[T] {
	return Point[T]{
		r.Max.X - r.Min.X,
		r.Max.Y - r.Min.Y,
	}
}

func (r Rect[T]) Add(p Point[T]) Rect[T] {
	return Rect[T]{
		Point[T]{r.Min.X + p.X, r.Min.Y + p.Y},
		Point[T]{r.Max.X + p.X, r.Max.Y + p.Y},
	}
}

func (r Rect[T]) Sub(p Point[T]) Rect[T] {
	return Rect[T]{
		Point[T]{r.Min.X - p.X, r.Min.Y - p.Y},
		Point[T]{r.Max.X - p.X, r.Max.Y - p.Y},
	}
}

func (r Rect[T]) Inset(n T) Rect[T] {
	if r.Dx() < 2*n {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += n
		r.Max.X -= n
	}
	if r.Dy() < 2*n {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += n
		r.Max.Y -= n
	}
	return r
}

func (r Rect[T]) Intersect(s Rect[T]) Rect[T] {
	if r.Min.X < s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y < s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X > s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y > s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	if r.Empty() {
		return Rect[T]{}
	}
	return r
}

func (r Rect[T]) Union(s Rect[T]) Rect[T] {
	if r.Empty() {
		return s
	}
	if s.Empty() {
		return r
	}
	if r.Min.X > s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y > s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X < s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y < s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	return r
}

func (r Rect[T]) Empty() bool {
	return r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y
}

func (r Rect[T]) Eq(s Rect[T]) bool {
	return r == s || r.Empty() && s.Empty()
}

func (r Rect[T]) Overlaps(s Rect[T]) bool {
	return !r.Empty() && !s.Empty() &&
		r.Min.X < s.Max.X && s.Min.X < r.Max.X &&
		r.Min.Y < s.Max.Y && s.Min.Y < r.Max.Y
}

func (r Rect[T]) In(s Rect[T]) bool {
	if r.Empty() {
		return true
	}
	return s.Min.X <= r.Min.X && r.Max.X <= s.Max.X &&
		s.Min.Y <= r.Min.Y && r.Max.Y <= s.Max.Y
}

func (r Rect[T]) Canon() Rect[T] {
	if r.Max.X < r.Min.X {
		r.Min.X, r.Max.X = r.Max.X, r.Min.X
	}
	if r.Max.Y < r.Min.Y {
		r.Min.Y, r.Max.Y = r.Max.Y, r.Min.Y
	}
	return r
}

// Center returns the point at the middle of r.
func (r Rect[T]) Center() Point[T] {
	return r.Min.Add(r.Max).Div(2)
}

// Align returns a new rectangle with the same dimensions as r but
// with a center point at p.
func (r Rect[T]) Align(p Point[T]) Rect[T] {
	hs := r.Size().Div(2)
	return Rt(
		p.X-hs.X,
		p.Y-hs.Y,
		p.X+hs.X,
		p.Y+hs.Y,
	)
}

// ClosestIn returns r shifted to be inside of s at the closest
// possible point to its starting position. If r is already entirely
// inside of s, r is returned unchanged. If r can not fit entirely
// inside of s, the zero Rect is returned.
func (r Rect[T]) ClosestIn(s Rect[T]) Rect[T] {
	if (r.Dx() > s.Dx()) || (r.Dy() > s.Dy()) {
		return Rect[T]{}
	}

	r = r.Canon()
	s = s.Canon()

	switch {
	case r.Min.X < s.Min.X:
		r.Max.X += s.Min.X - r.Min.X
		r.Min.X = s.Min.X
	case r.Max.X > s.Max.X:
		r.Min.X += s.Max.X - r.Max.X
		r.Max.X = s.Max.X
	}
	switch {
	case r.Min.Y < s.Min.Y:
		r.Max.Y += s.Min.Y - r.Min.Y
		r.Min.Y = s.Min.Y
	case r.Max.Y > s.Max.Y:
		r.Min.Y += s.Max.Y - r.Max.Y
		r.Max.Y = s.Max.Y
	}

	return r
}

func (r Rect[T]) Resize(size Point[T]) Rect[T] {
	return Rect[T]{Min: r.Min, Max: r.Min.Add(size)}
}

func (r Rect[T]) At(x, y T) color.Color {
	if (Point[T]{x, y}).In(r) {
		return color.Opaque
	}
	return color.Transparent
}

func (r Rect[T]) RGBA64At(x, y T) color.RGBA64 {
	if (Point[T]{x, y}).In(r) {
		return color.RGBA64{0xffff, 0xffff, 0xffff, 0xffff}
	}
	return color.RGBA64{}
}

func (r Rect[T]) Bounds() Rect[T] {
	return r
}

func (r Rect[T]) ColorModel() color.Model {
	return color.Alpha16Model
}

func (r Rect[T]) IsZero() bool {
	return r.Min.IsZero() && r.Max.IsZero()
}

func (r Rect[T]) ImageRect() image.Rectangle {
	return image.Rectangle{
		Min: r.Min.ImagePoint(),
		Max: r.Max.ImagePoint(),
	}
}
