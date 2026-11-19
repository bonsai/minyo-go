package seed

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

type spotDataset struct {
	Records []spotRecord `json:"records"`
}

type spotRecord struct {
	ID              string   `json:"id"`
	Prefecture      string   `json:"prefecture"`
	Municipality    string   `json:"municipality"`
	Name            string   `json:"name"`
	SongTitle       string   `json:"song_title"`
	SongType        string   `json:"song_type"`
	RelationType    string   `json:"relation_type"`
	EvidenceSummary string   `json:"evidence_summary"`
	LocationText    string   `json:"location_text"`
	Lat             *float64 `json:"lat"`
	Lon             *float64 `json:"lon"`
	SourceURL       string   `json:"source_url"`
	SourceType      string   `json:"source_type"`
	EvidenceLevel   string   `json:"evidence_level"`
	Status          string   `json:"status"`
}

// ImportSpots idempotently imports the verified spot ledger. Null coordinates
// remain NULL; the importer never infers or geocodes a check-in position.
func ImportSpots(database *sql.DB, filename string) (int, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return 0, fmt.Errorf("read spot dataset: %w", err)
	}
	var dataset spotDataset
	if err := json.Unmarshal(raw, &dataset); err != nil {
		return 0, fmt.Errorf("parse spot dataset: %w", err)
	}

	tx, err := database.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin spot import: %w", err)
	}
	defer tx.Rollback()

	const query = `INSERT INTO spots (
		id, prefecture, municipality, name, song_title, song_type,
		relation_type, evidence_summary, location_text, lat, lon,
		source_url, source_type, evidence_level, status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		prefecture=excluded.prefecture,
		municipality=excluded.municipality,
		name=excluded.name,
		song_title=excluded.song_title,
		song_type=excluded.song_type,
		relation_type=excluded.relation_type,
		evidence_summary=excluded.evidence_summary,
		location_text=excluded.location_text,
		lat=excluded.lat,
		lon=excluded.lon,
		source_url=excluded.source_url,
		source_type=excluded.source_type,
		evidence_level=excluded.evidence_level,
		status=excluded.status`

	for _, spot := range dataset.Records {
		if spot.ID == "" || spot.Name == "" || spot.SongTitle == "" {
			return 0, fmt.Errorf("invalid spot record: id, name and song_title are required")
		}
		if _, err := tx.Exec(query,
			spot.ID, spot.Prefecture, spot.Municipality, spot.Name,
			spot.SongTitle, spot.SongType, spot.RelationType,
			spot.EvidenceSummary, spot.LocationText, spot.Lat, spot.Lon,
			spot.SourceURL, spot.SourceType, spot.EvidenceLevel, spot.Status,
		); err != nil {
			return 0, fmt.Errorf("import spot %s: %w", spot.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit spot import: %w", err)
	}
	return len(dataset.Records), nil
}
