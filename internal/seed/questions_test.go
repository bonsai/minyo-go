package seed

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bonsai/minyo-go/internal/db"
)

func TestImportQuestionsRequiresThreePerSpotAndIsIdempotent(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "quiz.db"))
	if err != nil { t.Fatal(err) }
	defer database.Close()
	_, err = database.Exec(`INSERT INTO spots(id,prefecture,municipality,name,song_title,song_type,relation_type,evidence_summary,location_text,source_url,source_type,evidence_level,status)
		VALUES('spot-q','和歌山県','海南市','歌碑','歌','folk_song','song_monument','根拠','海南市','https://example.test','test','official_documented','verified')`)
	if err != nil { t.Fatal(err) }
	file := filepath.Join(t.TempDir(), "questions.json")
	data := `{"questions":[
		{"key":"q1","spot_id":"spot-q","prompt":"Q1","choices":["A","B"],"answer_index":0,"sort_order":1},
		{"key":"q2","spot_id":"spot-q","prompt":"Q2","choices":["A","B"],"answer_index":1,"sort_order":2},
		{"key":"q3","spot_id":"spot-q","prompt":"Q3","choices":["A","B"],"answer_index":0,"sort_order":3}
	]}`
	if err := os.WriteFile(file, []byte(data), 0600); err != nil { t.Fatal(err) }
	for i := 0; i < 2; i++ {
		count, err := ImportQuestions(database, file)
		if err != nil { t.Fatal(err) }
		if count != 3 { t.Fatalf("imported %d, want 3", count) }
	}
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM questions WHERE spot_id='spot-q'").Scan(&count); err != nil { t.Fatal(err) }
	if count != 3 { t.Fatalf("question count=%d, want 3", count) }
}


func TestRepositoryQuizDatasetImportsForAllVerifiedSpots(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "repository-quizzes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	spotsPath := filepath.Join("..", "..", "spots-verified.json")
	quizzesPath := filepath.Join("..", "..", "quizzes.json")
	spotCount, err := ImportSpots(database, spotsPath)
	if err != nil {
		t.Fatalf("import repository spots: %v", err)
	}
	if spotCount != 16 {
		t.Fatalf("imported spots=%d, want 16", spotCount)
	}
	questionCount, err := ImportQuestions(database, quizzesPath)
	if err != nil {
		t.Fatalf("import repository quizzes: %v", err)
	}
	if questionCount != 48 {
		t.Fatalf("imported questions=%d, want 48", questionCount)
	}

	var activeCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM questions WHERE active=1").Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if activeCount != 48 {
		t.Fatalf("active questions=%d, want 48", activeCount)
	}

	rows, err := database.Query("SELECT s.id, COUNT(q.id) FROM spots s LEFT JOIN questions q ON q.spot_id=s.id AND q.active=1 GROUP BY s.id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	checkedSpots := 0
	for rows.Next() {
		var spotID string
		var count int
		if err := rows.Scan(&spotID, &count); err != nil {
			t.Fatal(err)
		}
		checkedSpots++
		if count != 3 {
			t.Errorf("spot %s has %d active questions, want 3", spotID, count)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if checkedSpots != 16 {
		t.Fatalf("checked spots=%d, want 16", checkedSpots)
	}
}
