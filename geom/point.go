package geom

import (
	"image"

	"golang.org/x/exp/constraints"
)

type Point[T constraints.Integer | constraints.Float] struct {
	X, Y T
}

func Pt[T constraints.Integer | constraints.Float](X, Y T) Point[T] {
	return Point[T]{X, Y}
}

func PConv[Out constraints.Integer | constraints.Float, In constraints.Integer | constraints.Float](p Point[In]) Point[Out] {
	return Pt(Out(p.X), Out(p.Y))
}

func (p Point[T]) Add(q Point[T]) Point[T] {
	return Point[T]{p.X + q.X, p.Y + q.Y}
}

func (p Point[T]) Sub(q Point[T]) Point[T] {
	return Point[T]{p.X - q.X, p.Y - q.Y}
}

func (p Point[T]) Mul(k T) Point[T] {
	return Point[T]{p.X * k, p.Y * k}
}

func (p Point[T]) Div(k T) Point[T] {
	return Point[T]{p.X / k, p.Y / k}
}

func (p Point[T]) In(r Rect[T]) bool {
	return r.Min.X <= p.X && p.X < r.Max.X &&
		r.Min.Y <= p.Y && p.Y < r.Max.Y
}

func Mod[T constraints.Integer](p Point[T], r Rect[T]) Point[T] {
	w, h := r.Dx(), r.Dy()
	p = p.Sub(r.Min)
	p.X = p.X % w
	if p.X < 0 {
		p.X += w
	}
	p.Y = p.Y % h
	if p.Y < 0 {
		p.Y += h
	}
	return p.Add(r.Min)
}

func (p Point[T]) IsZero() bool {
	return (p.X == 0) && (p.Y == 0)
}

func (p Point[T]) ImagePoint() image.Point {
	return image.Pt(int(p.X), int(p.Y))
}

func Min[T constraints.Integer | constraints.Float](points ...Point[T]) Point[T] {
	r := points[0]
	for _, p := range points[1:] {
		if p.X < r.X {
			r.X = p.X
		}
		if p.Y < r.Y {
			r.Y = p.Y
		}
	}
	return r
}

func Max[T constraints.Integer | constraints.Float](points ...Point[T]) Point[T] {
	r := points[0]
	for _, p := range points[1:] {
		if p.X > r.X {
			r.X = p.X
		}
		if p.Y > r.Y {
			r.Y = p.Y
		}
	}
	return r
}
