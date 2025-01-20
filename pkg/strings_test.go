package pkg

import (
	"strconv"
	"testing"
)

func TestNumberToK(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{999, "999"},         // Less than 1000
		{1000, "1.0K"},       // Exactly 1000
		{1500, "1.5K"},       // Simple thousands
		{25000, "25.0K"},     // Larger thousands
		{1000000, "1000.0K"}, // Very large numbers
		{0, "0"},             // Edge case: zero
		{-500, "-500"},       // Negative numbers (no conversion)
		{-1500, "-1500"},     // Negative thousand (not supported)
	}

	for _, test := range tests {
		t.Run(
			"Input: "+strconv.Itoa(test.input),
			func(t *testing.T) {
				result := NumberToK(test.input)
				if result != test.expected {
					t.Errorf("ConvertToK(%d) = %s; expected %s", test.input, result, test.expected)
				}
			},
		)
	}
}
