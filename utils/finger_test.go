package utils

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/viper"
)

// TestFingerSummaryAvoidsDuplicateRoot ensures the homepage is fetched once while still crawling links/icons.
func TestFingerSummaryAvoidsDuplicateRoot(t *testing.T) {
	var rootHits, iconHits, pageHits, fallbackHits int32
	server := mustTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	oldClient, oldNoRedirect := Client, ClientNoRedirect
	Client, ClientNoRedirect = server.Client(), server.Client()
	t.Cleanup(func() {
		Client, ClientNoRedirect = oldClient, oldNoRedirect
	})

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

// TestSpiderFollowsRelativeJSUnderSubPath ensures assets referenced with relative paths under subdirectories are crawled.
func TestSpiderFollowsRelativeJSUnderSubPath(t *testing.T) {
	var jsHits int32
	server := mustTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/console/":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><script src="./static/app.js"></script></body></html>`))
		case "/console/static/app.js":
			atomic.AddInt32(&jsHits, 1)
			w.Header().Set("Content-Type", "application/javascript")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`const api="/api/secret";`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Use test HTTP client
	oldClient, oldNoRedirect := Client, ClientNoRedirect
	Client, ClientNoRedirect = server.Client(), server.Client()
	t.Cleanup(func() {
		Client, ClientNoRedirect = oldClient, oldNoRedirect
	})

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

	summary := FingerSummary(server.URL+"/console/", 2, db)
	if summary.ApiCount != 1 {
		t.Fatalf("api count = %d, want 1", summary.ApiCount)
	}
	if hits := atomic.LoadInt32(&jsHits); hits == 0 {
		t.Fatalf("js hits = %d, want >0", hits)
	}
}

// TestSpiderExtractsSensitiveFromJS ensures JS assets are scanned for sensitive info (Bearer/password).
func TestSpiderExtractsSensitiveFromJS(t *testing.T) {
	var jsHits int32
	server := mustTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><script src="/app.js"></script></body></html>`))
		case "/app.js":
			atomic.AddInt32(&jsHits, 1)
			w.Header().Set("Content-Type", "application/javascript")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`Authorization: "Bearer AAAAABBBBB"; password: "p@ssw0rd"`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	oldClient, oldNoRedirect := Client, ClientNoRedirect
	Client, ClientNoRedirect = server.Client(), server.Client()
	t.Cleanup(func() {
		Client, ClientNoRedirect = oldClient, oldNoRedirect
	})

	tmpDir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	_ = os.Chdir(tmpDir)

	dbPath := filepath.Join(tmpDir, "spider.db")
	db, err := InitSpiderDB(dbPath)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()
	SetSpiderDB(db)

	viper.Set("max-body-bytes", 1024)

	summary := FingerSummary(server.URL, 2, db)
	if summary.ApiCount != 0 {
		t.Fatalf("api count = %d, want 0", summary.ApiCount)
	}
	if hits := atomic.LoadInt32(&jsHits); hits == 0 {
		t.Fatalf("js hits = %d, want >0", hits)
	}

	hits, err := LoadSensitiveHits(db)
	if err != nil {
		t.Fatalf("load sensitive hits: %v", err)
	}
	var foundBearer, foundPassword bool
	for _, h := range hits {
		if strings.Contains(h.Content, "Bearer AAAAABBBBB") {
			foundBearer = true
		}
		if strings.Contains(strings.ToLower(h.Content), "p@ssw0rd") {
			foundPassword = true
		}
	}
	if !foundBearer || !foundPassword {
		t.Fatalf("sensitive hits missing bearer/password, got %+v", hits)
	}
}
