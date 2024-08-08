package pkg

func Capped[T any](cap int, slice []T) []T {
	capped := cap
	if capped > len(slice) {
		capped = len(slice)
	}
	return slice[:capped]
}
