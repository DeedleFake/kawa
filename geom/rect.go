package geom

import (
	"image"

	"golang.org/x/exp/constraints"
)

// A Rect contains the points with Min.X <= X < Max.X, Min.Y <= Y < Max.Y. It
// is well-formed if Min.X <= Max.X and likewise for Y. Points are always
// well-formed. A rectangle's methods always return well-formed outputs for
// well-formed inputs.
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

func FromImageRect(r image.Rectangle) Rect[int] {
	return Rect[int]{
		Min: FromImagePoint(r.Min),
		Max: FromImagePoint(r.Max),
	}
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
	return r.Inset2(Pt(n, n))
}

func (r Rect[T]) Inset2(n Point[T]) Rect[T] {
	if r.Dx() < 2*n.X {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += n.X
		r.Max.X -= n.X
	}
	if r.Dy() < 2*n.Y {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += n.Y
		r.Max.Y -= n.Y
	}
	return r
}

func (r Rect[T]) Pad(top, bottom, left, right T) Rect[T] {
	r = r.Canon()
	r.Min.X += left
	r.Max.X -= right
	r.Min.Y += top
	r.Max.Y -= bottom
	if r.Dx() < 0 {
		r.Max.X = r.Min.X
	}
	if r.Dy() < 0 {
		r.Max.Y = r.Min.Y
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

func (r Rect[T]) FitTo(size Point[T]) Rect[T] {
	aspect := r.Aspect()
	r.Max = r.Min.Add(size)
	return r.WithAspect(aspect)
}

func (r Rect[T]) Aspect() float64 {
	return float64(r.Dx()) / float64(r.Dy())
}

func (r Rect[T]) WithAspect(aspect float64) Rect[T] {
	if r.Aspect() > aspect {
		return r.Resize(Pt(T(float64(r.Dy())*aspect), r.Dy()))
	}
	return r.Resize(Pt(r.Dx(), T(float64(r.Dx())/aspect)))
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
