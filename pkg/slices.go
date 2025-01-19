package pkg

// nolint: revive
func Capped[T any](cap int, slice []T) []T {
	capped := cap
	if capped > len(slice) {
		capped = len(slice)
	}
	return slice[:capped]
}

func NonStreakZero(streaks []int, streak int, minimum int) bool {
	// check if last three elements are over a minimum
	if len(streaks) < streak {
		return false
	}
	for i := 0; i < streak; i++ {
		if streaks[len(streaks)-1-i] < minimum {
			return false
		}
	}
	return true
}
