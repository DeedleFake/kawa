package geom

import (
	"image"
	"image/color"

	"golang.org/x/exp/constraints"
)

type Rect[T constraints.Integer | constraints.Float] struct {
	Min, Max Point[T]
}

func Rt[T constraints.Integer | constraints.Float](x0, y0, x1, y1 T) Rect[T] {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return Rect[T]{Point[T]{x0, y0}, Point[T]{x1, y1}}
}

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

func (r Rect[T]) Center() Point[T] {
	return r.Min.Add(r.Max).Div(2)
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
