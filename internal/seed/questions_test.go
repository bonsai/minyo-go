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
