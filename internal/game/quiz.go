package game

// ValidateAnswerIDs checks that a submission contains each expected question exactly once.
func ValidateAnswerIDs(expectedIDs []int64, submittedIDs []int64) bool {
	if len(expectedIDs) != 3 || len(submittedIDs) != len(expectedIDs) {
		return false
	}
	expected := make(map[int64]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		if _, exists := expected[id]; exists {
			return false
		}
		expected[id] = struct{}{}
	}
	seen := make(map[int64]struct{}, len(submittedIDs))
	for _, id := range submittedIDs {
		if _, ok := expected[id]; !ok {
			return false
		}
		if _, duplicate := seen[id]; duplicate {
			return false
		}
		seen[id] = struct{}{}
	}
	return len(seen) == 3
}

// Passed reports whether all three quiz answers are correct.
func Passed(correctCount, totalCount int) bool {
	return totalCount == 3 && correctCount == 3
}
