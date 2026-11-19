package seed

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/bonsai/minyo-go/internal/db"
)

func TestImportSpotsIsIdempotentAndKeepsNullCoordinates(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "seed.db"))
	if err != nil { t.Fatal(err) }
	defer database.Close()
	file := filepath.Join(t.TempDir(), "spots.json")
	body := `{"records":[{"id":"spot-a","prefecture":"和歌山県","municipality":"海南市","name":"テスト歌碑","song_title":"テストの歌","song_type":"folk_song","relation_type":"song_monument","evidence_summary":"根拠","location_text":"海南市","lat":null,"lon":null,"source_url":"https://example.test","source_type":"test","evidence_level":"official_documented","status":"needs_safe_checkin_point"}]}`
	if err := os.WriteFile(file, []byte(body), 0600); err != nil { t.Fatal(err) }
	for i := 0; i < 2; i++ {
		count, err := ImportSpots(database, file)
		if err != nil { t.Fatal(err) }
		if count != 1 { t.Fatalf("imported %d records, want 1", count) }
	}
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM spots WHERE id='spot-a'").Scan(&count); err != nil { t.Fatal(err) }
	if count != 1 { t.Fatalf("row count=%d, want 1", count) }
	var lat, lon sql.NullFloat64
	if err := database.QueryRow("SELECT lat, lon FROM spots WHERE id='spot-a'").Scan(&lat, &lon); err != nil { t.Fatal(err) }
	if lat.Valid || lon.Valid { t.Fatal("null coordinates must remain NULL") }
}
