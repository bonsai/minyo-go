package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientCannotSubmitAuthoritativePassFlag(t *testing.T) {
	server, closeDB := testServer(t, true)
	defer closeDB()

	body := `{"user_id":"forged-user","spot_id":"spot-1","answers":[{"question_id":1,"answer_index":0},{"question_id":2,"answer_index":1},{"question_id":3,"answer_index":2}],"test_mode":true,"passed":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/attempts", strings.NewReader(body))
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("forged client result status=%d, want %d; body=%s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	var count int
	if err := server.db.QueryRow("SELECT COUNT(*) FROM stamps WHERE user_id='forged-user' AND spot_id='spot-1'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("forged client result created %d stamps, want 0", count)
	}
}
