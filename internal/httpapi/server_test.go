package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/bonsai/minyo-go/internal/db"
)

func testServer(t *testing.T, allowTestMode bool) (*Server, func()) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil { t.Fatal(err) }
	_, err = database.Exec(`INSERT INTO spots (
		id, prefecture, municipality, name, song_title, song_type, relation_type,
		evidence_summary, location_text, lat, lon, source_url, source_type, evidence_level, status
	) VALUES ('spot-1','大阪府','大阪市','テスト歌碑','テストの歌','folk_song','song_monument',
		'根拠','大阪市',34.6937,135.5023,'https://example.test','test','official_documented','verified')`)
	if err != nil { t.Fatal(err) }
	for i := 0; i < 3; i++ {
		_, err = database.Exec("INSERT INTO questions(spot_id,prompt,choices_json,answer_index,sort_order) VALUES('spot-1',?,?,?,?)",
			"質問", `["A","B","C"]`, i, i)
		if err != nil { t.Fatal(err) }
	}
	return New(database, allowTestMode), func() { database.Close() }
}

func postAttempt(t *testing.T, server *Server, answers []submittedAnswer, testMode bool) *httptest.ResponseRecorder {
	t.Helper()
	payload := attemptRequest{UserID:"user-1", SpotID:"spot-1", Answers:answers, TestMode:testMode}
	body, err := json.Marshal(payload)
	if err != nil { t.Fatal(err) }
	req := httptest.NewRequest(http.MethodPost, "/api/attempts", bytes.NewReader(body))
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	return res
}

func TestQuizDoesNotExposeAnswerKey(t *testing.T) {
	server, closeDB := testServer(t, true)
	defer closeDB()
	req := httptest.NewRequest(http.MethodGet, "/api/spots/spot-1/quiz", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK { t.Fatalf("status = %d, body=%s", res.Code, res.Body.String()) }
	if bytes.Contains(res.Body.Bytes(), []byte("answer_index")) { t.Fatal("quiz response leaked answer_index") }
}

func TestStampAwardIsIdempotent(t *testing.T) {
	server, closeDB := testServer(t, true)
	defer closeDB()
	answers := []submittedAnswer{{QuestionID:1,AnswerIndex:0},{QuestionID:2,AnswerIndex:1},{QuestionID:3,AnswerIndex:2}}
	first := postAttempt(t, server, answers, true)
	if first.Code != http.StatusOK { t.Fatalf("first status=%d body=%s", first.Code, first.Body.String()) }
	if !bytes.Contains(first.Body.Bytes(), []byte(`"stamp_awarded":true`)) { t.Fatalf("first attempt did not award stamp: %s", first.Body.String()) }
	second := postAttempt(t, server, answers, true)
	if second.Code != http.StatusOK { t.Fatalf("second status=%d body=%s", second.Code, second.Body.String()) }
	if !bytes.Contains(second.Body.Bytes(), []byte(`"stamp_awarded":false`)) { t.Fatalf("duplicate stamp was awarded: %s", second.Body.String()) }
}

func TestWrongAnswerDoesNotAwardStamp(t *testing.T) {
	server, closeDB := testServer(t, true)
	defer closeDB()
	answers := []submittedAnswer{{QuestionID:1,AnswerIndex:2},{QuestionID:2,AnswerIndex:1},{QuestionID:3,AnswerIndex:2}}
	res := postAttempt(t, server, answers, true)
	if res.Code != http.StatusOK { t.Fatalf("status=%d body=%s", res.Code, res.Body.String()) }
	if !bytes.Contains(res.Body.Bytes(), []byte(`"passed":false`)) { t.Fatalf("attempt unexpectedly passed: %s", res.Body.String()) }
	if !bytes.Contains(res.Body.Bytes(), []byte(`"retry_allowed":true`)) { t.Fatalf("retry not allowed: %s", res.Body.String()) }
}

func TestTestModeMustBeEnabledByServer(t *testing.T) {
	server, closeDB := testServer(t, false)
	defer closeDB()
	answers := []submittedAnswer{{QuestionID:1,AnswerIndex:0},{QuestionID:2,AnswerIndex:1},{QuestionID:3,AnswerIndex:2}}
	res := postAttempt(t, server, answers, true)
	if res.Code != http.StatusForbidden { t.Fatalf("status=%d, want %d", res.Code, http.StatusForbidden) }
}

func TestThirtyKilometerRadiusIsEnforced(t *testing.T) {
	server, closeDB := testServer(t, false)
	defer closeDB()
	answers := []submittedAnswer{{QuestionID:1,AnswerIndex:0},{QuestionID:2,AnswerIndex:1},{QuestionID:3,AnswerIndex:2}}
	payload := attemptRequest{UserID:"near-user", SpotID:"spot-1", Answers:answers, Latitude:floatPtr(34.6937), Longitude:floatPtr(135.5023)}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/attempts", bytes.NewReader(body))
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK { t.Fatalf("nearby attempt status=%d body=%s", res.Code, res.Body.String()) }

	payload.UserID = "far-user"
	payload.Latitude = floatPtr(35.5)
	payload.Longitude = floatPtr(139.0)
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest(http.MethodPost, "/api/attempts", bytes.NewReader(body))
	res = httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden { t.Fatalf("far attempt status=%d, want %d", res.Code, http.StatusForbidden) }
}

func TestNormalModeRejectsUnverifiedSpotCoordinates(t *testing.T) {
	server, closeDB := testServer(t, false)
	defer closeDB()
	if _, err := server.db.Exec("UPDATE spots SET lat=NULL, lon=NULL WHERE id='spot-1'"); err != nil { t.Fatal(err) }
	answers := []submittedAnswer{{QuestionID:1,AnswerIndex:0},{QuestionID:2,AnswerIndex:1},{QuestionID:3,AnswerIndex:2}}
	payload := attemptRequest{UserID:"null-coord-user", SpotID:"spot-1", Answers:answers, Latitude:floatPtr(34.6937), Longitude:floatPtr(135.5023)}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/attempts", bytes.NewReader(body))
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusConflict { t.Fatalf("unverified spot status=%d, want %d", res.Code, http.StatusConflict) }
}

func floatPtr(value float64) *float64 { return &value }
