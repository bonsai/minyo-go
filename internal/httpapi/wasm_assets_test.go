package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bonsai/minyo-go/internal/db"
)

func TestWASMAssetsAreServedWithBrowserMIMETypes(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test source")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	wasmPath := filepath.Join(repoRoot, "wasm", "minyo-go.wasm")
	if _, err := os.Stat(wasmPath); err != nil {
		t.Skip("WASM bundle is not built; run scripts/build-wasm.sh first")
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	database, err := db.Open(filepath.Join(t.TempDir(), "wasm-assets.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	handler := New(database).Handler()

	cases := []struct {
		path string
		wantType string
	}{
		{"/wasm/minyo-go.wasm", "application/wasm"},
		{"/wasm/wasm_exec.js", "text/javascript"},
		{"/wasm/wasm-loader.js", "text/javascript"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("GET %s status=%d body=%s", tc.path, res.Code, res.Body.String())
			}
			if got := res.Header().Get("Content-Type"); !strings.HasPrefix(got, tc.wantType) {
				t.Fatalf("GET %s Content-Type=%q, want prefix %q", tc.path, got, tc.wantType)
			}
		})
	}
}
