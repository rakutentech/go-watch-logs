package pkg

import (
	"fmt"
	"sort"
	"strings"
)

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

func UniqueStrings(input []string) []string {
	uniqueMap := make(map[string]struct{})
	for _, str := range input {
		uniqueMap[str] = struct{}{}
	}
	uniqueList := make([]string, 0, len(uniqueMap))
	for str := range uniqueMap {
		uniqueList = append(uniqueList, str)
	}
	return uniqueList
}

func ToCSV(input []string) string {
	result := ""
	for i, str := range input {
		if i > 0 {
			result += ", "
		}
		result += str
	}
	return result
}

func OrderedAsc(slice map[string]int) string {
	type kv struct {
		Key   string
		Value int
	}
	// 1. Create a slice of key-value structs from the map.
	ss := make([]kv, 0, len(slice))
	for k, v := range slice {
		ss = append(ss, kv{k, v})
	}

	// 2. Sort the slice based on the Value field in descending order.
	//    To sort in ascending order, change the '>' to '<'.
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	// 3. Build the final string efficiently using strings.Builder.
	var sb strings.Builder
	for _, pair := range ss {
		// Use fmt.Fprintf to write the formatted string directly to the builder.
		fmt.Fprintf(&sb, "%s: %d, ", pair.Key, pair.Value)
	}

	// Remove the trailing comma if necessary.
	result := sb.String()
	if len(result) > 0 {
		result = result[:len(result)-2] // Remove trailing ", "
	}

	return result
}
