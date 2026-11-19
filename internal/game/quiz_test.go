package game

import "testing"

func TestValidateAnswerIDs(t *testing.T) {
	tests := []struct {
		name string
		expected []int64
		submitted []int64
		want bool
	}{
		{"valid", []int64{10, 20, 30}, []int64{30, 10, 20}, true},
		{"missing", []int64{10, 20, 30}, []int64{10, 20}, false},
		{"duplicate", []int64{10, 20, 30}, []int64{10, 20, 20}, false},
		{"unknown", []int64{10, 20, 30}, []int64{10, 20, 40}, false},
		{"duplicate expected", []int64{10, 10, 30}, []int64{10, 10, 30}, false},
		{"not three", []int64{10, 20}, []int64{10, 20}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateAnswerIDs(tt.expected, tt.submitted); got != tt.want {
				t.Fatalf("ValidateAnswerIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPassed(t *testing.T) {
	for _, tt := range []struct{ correct, total int; want bool }{
		{3, 3, true}, {2, 3, false}, {3, 4, false}, {0, 3, false},
	} {
		if got := Passed(tt.correct, tt.total); got != tt.want {
			t.Errorf("Passed(%d, %d) = %v, want %v", tt.correct, tt.total, got, tt.want)
		}
	}
}
