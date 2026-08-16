package sets

func Difference[T comparable](a, b []T) []T {
	inB := make(map[T]struct{}, len(b))
	for _, v := range b {
		inB[v] = struct{}{}
	}

	var result []T
	for _, v := range a {
		if _, ok := inB[v]; !ok {
			result = append(result, v)
		}
	}
	return result
}
