package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"math"
)

type Server struct {
	db *sql.DB
	allowTestMode bool
}

// New creates an API server. Test mode must be explicitly enabled by the operator.
func New(database *sql.DB, allowTestMode ...bool) *Server {
	allow := len(allowTestMode) > 0 && allowTestMode[0]
	return &Server{db: database, allowTestMode: allow}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" { http.NotFound(w, r); return }
		http.ServeFile(w, r, "map.html")
	})
	mux.HandleFunc("GET /map.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "map.html")
	})
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/spots", s.listSpots)
	mux.HandleFunc("GET /api/spots/{id}/quiz", s.getQuiz)
	mux.HandleFunc("POST /api/attempts", s.submitAttempt)
	mux.HandleFunc("GET /api/users/{id}/stamps", s.getStamps)
	mux.HandleFunc("GET /api/users/{id}/progress", s.getProgress)
	return withJSONHeaders(mux)
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.db.PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listSpots(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, prefecture, municipality, name, song_title, song_type,
		       relation_type, location_text, lat, lon, status
		FROM spots ORDER BY prefecture, municipality, name`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not list spots"})
		return
	}
	defer rows.Close()

	type spot struct {
		ID string `json:"id"`
		Prefecture string `json:"prefecture"`
		Municipality string `json:"municipality"`
		Name string `json:"name"`
		SongTitle string `json:"song_title"`
		SongType string `json:"song_type"`
		RelationType string `json:"relation_type"`
		LocationText string `json:"location_text"`
		Lat *float64 `json:"lat"`
		Lon *float64 `json:"lon"`
		Status string `json:"status"`
	}
	result := make([]spot, 0)
	for rows.Next() {
		var item spot
		if err := rows.Scan(&item.ID, &item.Prefecture, &item.Municipality,
			&item.Name, &item.SongTitle, &item.SongType, &item.RelationType,
			&item.LocationText, &item.Lat, &item.Lon, &item.Status); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read spots"})
			return
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read spots"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"spots": result, "count": len(result)})
}

type quizQuestion struct {
	ID int64 `json:"id"`
	Prompt string `json:"prompt"`
	Choices []string `json:"choices"`
}

func (s *Server) getQuiz(w http.ResponseWriter, r *http.Request) {
	spotID := r.PathValue("id")
	var exists int
	if err := s.db.QueryRowContext(r.Context(), "SELECT 1 FROM spots WHERE id = ?", spotID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "spot not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not verify spot"})
		}
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, prompt, choices_json FROM questions
		WHERE spot_id = ? AND active = 1 ORDER BY sort_order, id LIMIT 4`, spotID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load quiz"})
		return
	}
	defer rows.Close()
	questions := make([]quizQuestion, 0, 3)
	for rows.Next() {
		var q quizQuestion
		var choices string
		if err := rows.Scan(&q.ID, &q.Prompt, &choices); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read quiz"})
			return
		}
		if err := json.Unmarshal([]byte(choices), &q.Choices); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid quiz choices"})
			return
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read quiz"})
		return
	}
	if len(questions) != 3 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "this spot does not have exactly three active questions yet"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"spot_id": spotID, "questions": questions})
}

type submittedAnswer struct {
	QuestionID int64 `json:"question_id"`
	AnswerIndex int `json:"answer_index"`
}
type attemptRequest struct {
	UserID string `json:"user_id"`
	SpotID string `json:"spot_id"`
	Answers []submittedAnswer `json:"answers"`
	Latitude *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	TestMode bool `json:"test_mode"`
}

func (s *Server) submitAttempt(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var request attemptRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	request.UserID = strings.TrimSpace(request.UserID)
	request.SpotID = strings.TrimSpace(request.SpotID)
	if request.UserID == "" || request.SpotID == "" || len(request.Answers) != 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id, spot_id and exactly three answers are required"})
		return
	}

	var spotLat, spotLon sql.NullFloat64
	if err := s.db.QueryRowContext(r.Context(), "SELECT lat, lon FROM spots WHERE id = ?", request.SpotID).Scan(&spotLat, &spotLon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "spot not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not verify spot"})
		}
		return
	}
	if request.TestMode {
		if !s.allowTestMode {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "test mode is disabled on this server"})
			return
		}
	} else {
		if request.Latitude == nil || request.Longitude == nil || *request.Latitude < -90 || *request.Latitude > 90 || *request.Longitude < -180 || *request.Longitude > 180 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid latitude and longitude are required"})
			return
		}
		if !spotLat.Valid || !spotLon.Valid {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "this spot has no field-verified check-in coordinates"})
			return
		}
		if haversineKm(*request.Latitude, *request.Longitude, spotLat.Float64, spotLon.Float64) > 30 {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "outside the 30 km GET radius"})
			return
		}
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not start attempt"})
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(r.Context(), "INSERT INTO users(id) VALUES (?) ON CONFLICT(id) DO NOTHING", request.UserID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not register user"})
		return
	}

	rows, err := tx.QueryContext(r.Context(), `
		SELECT id, answer_index FROM questions
		WHERE spot_id = ? AND active = 1 ORDER BY sort_order, id`, request.SpotID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load answers"})
		return
	}
	expected := make(map[int64]int, 3)
	for rows.Next() {
		var id int64
		var answer int
		if err := rows.Scan(&id, &answer); err != nil {
			rows.Close()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read answers"})
			return
		}
		expected[id] = answer
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read answers"})
		return
	}
	if len(expected) != 3 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "this spot does not have exactly three active questions yet"})
		return
	}

	seen := make(map[int64]bool, 3)
	correct := 0
	for _, answer := range request.Answers {
		expectedIndex, exists := expected[answer.QuestionID]
		if !exists || seen[answer.QuestionID] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "answers must contain each active question exactly once"})
			return
		}
		seen[answer.QuestionID] = true
		if answer.AnswerIndex == expectedIndex {
			correct++
		}
	}
	if len(seen) != 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "answers must contain each active question exactly once"})
		return
	}

	passed := correct == 3
	if _, err := tx.ExecContext(r.Context(), `
		INSERT INTO attempts(user_id, spot_id, correct_count, total_count, passed)
		VALUES (?, ?, ?, 3, ?)`, request.UserID, request.SpotID, correct, passed); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save attempt"})
		return
	}
	stampAwarded := false
	if passed {
		result, err := tx.ExecContext(r.Context(), `
			INSERT INTO stamps(user_id, spot_id) VALUES (?, ?)
			ON CONFLICT(user_id, spot_id) DO NOTHING`, request.UserID, request.SpotID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not award stamp"})
			return
		}
		affected, _ := result.RowsAffected()
		stampAwarded = affected == 1
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not complete attempt"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"correct_count": correct,
		"total_count": 3,
		"passed": passed,
		"stamp_awarded": stampAwarded,
		"stamp_already_owned": passed && !stampAwarded,
		"retry_allowed": !passed,
	})
}

func (s *Server) getStamps(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT st.spot_id, sp.name, sp.song_title, st.earned_at
		FROM stamps st JOIN spots sp ON sp.id = st.spot_id
		WHERE st.user_id = ? ORDER BY st.earned_at, st.id`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load stamps"})
		return
	}
	defer rows.Close()
	type stamp struct {
		SpotID string `json:"spot_id"`
		SpotName string `json:"spot_name"`
		SongTitle string `json:"song_title"`
		EarnedAt string `json:"earned_at"`
	}
	items := make([]stamp, 0)
	for rows.Next() {
		var item stamp
		if err := rows.Scan(&item.SpotID, &item.SpotName, &item.SongTitle, &item.EarnedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read stamps"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read stamps"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "stamps": items, "count": len(items)})
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRadians := func(degrees float64) float64 { return degrees * math.Pi / 180 }
	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

func (s *Server) getProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT sp.id, sp.name, sp.song_title, st.earned_at
		FROM spots sp
		LEFT JOIN stamps st ON st.spot_id = sp.id AND st.user_id = ?
		ORDER BY sp.prefecture, sp.municipality, sp.name`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load progress"})
		return
	}
	defer rows.Close()
	type progressItem struct {
		SpotID string `json:"spot_id"`
		SpotName string `json:"spot_name"`
		SongTitle string `json:"song_title"`
		Collected bool `json:"collected"`
		EarnedAt *string `json:"earned_at"`
	}
	items := make([]progressItem, 0)
	collected := 0
	for rows.Next() {
		var item progressItem
		if err := rows.Scan(&item.SpotID, &item.SpotName, &item.SongTitle, &item.EarnedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read progress"})
			return
		}
		item.Collected = item.EarnedAt != nil
		if item.Collected { collected++ }
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read progress"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"total_spots": len(items),
		"collected_spots": collected,
		"remaining_spots": len(items)-collected,
		"spots": items,
	})
}
