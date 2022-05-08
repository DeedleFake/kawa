package util

func FindFunc[E any](s []E, f func(E) bool) (e E, ok bool) {
	for _, e := range s {
		if f(e) {
			return e, true
		}
	}
	return e, false
}
