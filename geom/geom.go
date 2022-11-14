package geom

import "golang.org/x/exp/constraints"

// Scalar is a constraint for the types that geom types and functions
// can handle.
type Scalar interface {
	constraints.Integer | constraints.Float
}
