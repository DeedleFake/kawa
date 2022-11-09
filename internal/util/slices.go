package util

// Last returns the last element from the provided slices. In other
// words, if it is passed two slices and the second is empty, it will
// return the last item of the first. ok is only false if all of the
// provided slices are empty or no slices were provided.
func Last[T any](s ...[]T) (v T, ok bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if len(s[i]) > 0 {
			return s[i][len(s[i])-1], true
		}
	}
	return v, false
}

// Match returns a function that returns true if passed v. T must be
// a comparable type or the returned function will panic. T may be an
// interface type.
func Match[T any](v T) func(T) bool {
	return func(v2 T) bool {
		return any(v) == any(v2)
	}
}
