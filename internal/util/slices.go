package util

func Last[T any](s ...[]T) (v T, ok bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if len(s[i]) > 0 {
			return s[i][len(s[i])-1], true
		}
	}
	return v, false
}
