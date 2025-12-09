package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/spf13/viper"
)

// TestFingerSummaryAvoidsDuplicateRoot ensures the homepage is fetched once while still crawling links/icons.
func TestFingerSummaryAvoidsDuplicateRoot(t *testing.T) {
	var rootHits, iconHits, pageHits, fallbackHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			atomic.AddInt32(&rootHits, 1)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><head><link rel="icon" href="/favicon.ico"></head><body><a href="/page">link</a></body></html>`))
		case "/favicon.ico":
			atomic.AddInt32(&iconHits, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ico"))
		case "/page":
			atomic.AddInt32(&pageHits, 1)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body>page</body></html>"))
		case "/xxxxxx":
			atomic.AddInt32(&fallbackHits, 1)
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Use test HTTP client
	Client = server.Client()
	ClientNoRedirect = server.Client()

	// Temp workspace
	tmpDir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	_ = os.Chdir(tmpDir)

	// Setup DB
	dbPath := filepath.Join(tmpDir, "spider.db")
	db, err := InitSpiderDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()
	SetSpiderDB(db)

	// Avoid large bodies
	viper.Set("max-body-bytes", 1024)

	// Run fingerprint + spider (depth 2 to allow link crawl)
	summary := FingerSummary(server.URL, 2, db)
	if summary.Err != nil {
		t.Fatalf("finger summary error: %v", summary.Err)
	}

	if got := atomic.LoadInt32(&rootHits); got != 1 {
		t.Fatalf("root hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&iconHits); got != 1 {
		t.Fatalf("icon hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&fallbackHits); got != 1 {
		t.Fatalf("fallback hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&pageHits); got < 1 {
		t.Fatalf("page hits = %d, want >=1", got)
	}

	if summary.URL != server.URL {
		t.Fatalf("summary URL mismatch: %s", summary.URL)
	}
}
