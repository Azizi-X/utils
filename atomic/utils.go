package atomic

func packV[T any](s T) any {
	return s
}

func unpackV[T any](v any) T {
	s, _ := v.(T)
	return s
}
