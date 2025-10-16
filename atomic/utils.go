package atomic

type nocmp [0]func()

func packString(s string) any {
	return s
}

func unpackString(v any) string {
	s, _ := v.(string)
	return s
}

func packV[T any](s T) any {
	return s
}

func unpackV[T any](v any) T {
	s, _ := v.(T)
	return s
}
