package seed

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type questionDataset struct {
	Questions []questionRecord `json:"questions"`
}

type questionRecord struct {
	Key         string   `json:"key"`
	SpotID      string   `json:"spot_id"`
	Prompt      string   `json:"prompt"`
	Choices     []string `json:"choices"`
	AnswerIndex int      `json:"answer_index"`
	SortOrder   int      `json:"sort_order"`
}

// ImportQuestions imports an authored quiz dataset. Each represented spot must
// have exactly three questions; answer keys are stored server-side only.
func ImportQuestions(database *sql.DB, filename string) (int, error) {
	raw, err := os.ReadFile(filename)
	if err != nil { return 0, fmt.Errorf("read quiz dataset: %w", err) }
	var dataset questionDataset
	if err := json.Unmarshal(raw, &dataset); err != nil { return 0, fmt.Errorf("parse quiz dataset: %w", err) }
	if len(dataset.Questions) == 0 { return 0, fmt.Errorf("quiz dataset contains no questions") }

	perSpot := make(map[string]int)
	seen := make(map[string]bool)
	for _, q := range dataset.Questions {
		if q.Key == "" || q.SpotID == "" || q.Prompt == "" || len(q.Choices) < 2 {
			return 0, fmt.Errorf("each question requires key, spot_id, prompt and at least two choices")
		}
		if q.AnswerIndex < 0 || q.AnswerIndex >= len(q.Choices) {
			return 0, fmt.Errorf("question %s has an answer_index outside its choices", q.Key)
		}
		if seen[q.Key] { return 0, fmt.Errorf("duplicate question key: %s", q.Key) }
		seen[q.Key] = true
		perSpot[q.SpotID]++
	}
	for spotID, count := range perSpot {
		if count != 3 { return 0, fmt.Errorf("spot %s has %d questions; exactly 3 are required", spotID, count) }
	}

	tx, err := database.Begin()
	if err != nil { return 0, fmt.Errorf("begin quiz import: %w", err) }
	defer tx.Rollback()
	for _, q := range dataset.Questions {
		choices, err := json.Marshal(q.Choices)
		if err != nil { return 0, fmt.Errorf("encode choices for %s: %w", q.Key, err) }
		_, err = tx.Exec(`INSERT INTO questions(external_key, spot_id, prompt, choices_json, answer_index, sort_order, active)
			VALUES (?, ?, ?, ?, ?, ?, 1)
			ON CONFLICT(external_key) DO UPDATE SET
				spot_id=excluded.spot_id, prompt=excluded.prompt,
				choices_json=excluded.choices_json, answer_index=excluded.answer_index,
				sort_order=excluded.sort_order, active=1`,
			q.Key, q.SpotID, q.Prompt, string(choices), q.AnswerIndex, q.SortOrder)
		if err != nil { return 0, fmt.Errorf("import question %s: %w", q.Key, err) }
	}
	if err := tx.Commit(); err != nil { return 0, fmt.Errorf("commit quiz import: %w", err) }
	return len(dataset.Questions), nil
}
