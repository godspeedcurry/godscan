package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/godspeedcurry/godscan/common"
)

// TestIconDetectAutoPage ensures icon detection fetches page once and icon once.
func TestIconDetectAutoPage(t *testing.T) {
	var pageHits, iconHits int32
	iconBody := []byte("ico-content")

	srv := mustTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			atomic.AddInt32(&pageHits, 1)
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><link rel="icon" href="/favicon.ico"></head></html>`))
		case "/favicon.ico":
			atomic.AddInt32(&iconHits, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(iconBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldClient, oldNoRedirect := Client, ClientNoRedirect
	Client, ClientNoRedirect = srv.Client(), srv.Client()
	defer func() {
		Client, ClientNoRedirect = oldClient, oldNoRedirect
	}()

	fofa, hunter, err := IconDetectAuto(srv.URL)
	if err != nil {
		t.Fatalf("IconDetectAuto error: %v", err)
	}
	if fofa == "" || hunter == "" {
		t.Fatalf("expected non-empty hashes, got fofa=%q hunter=%q", fofa, hunter)
	}
	if got := atomic.LoadInt32(&pageHits); got != 1 {
		t.Fatalf("page hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&iconHits); got != 1 {
		t.Fatalf("icon hits = %d, want 1", got)
	}
}

// TestFingerScanRedirect ensures redirects are reported without following body processing.
func TestFingerScanRedirect(t *testing.T) {
	srv := mustTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dest", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	oldClient, oldNoRedirect := Client, ClientNoRedirect
	Client, ClientNoRedirect = srv.Client(), srv.Client()
	defer func() {
		Client, ClientNoRedirect = oldClient, oldNoRedirect
	}()

	res := FingerScan(srv.URL, http.MethodGet, false)
	if res.Status != http.StatusFound {
		t.Fatalf("status = %d, want 302", res.Status)
	}
	if res.Location == "" {
		t.Fatalf("expected redirect location, got empty")
	}
	if res.Finger != common.NoFinger {
		t.Fatalf("finger = %s, want %s", res.Finger, common.NoFinger)
	}
}

// TestFingerScanFollowRedirect ensures redirects are followed when requested.
func TestFingerScanFollowRedirect(t *testing.T) {
	srv := mustTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dest", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><title>dest</title></html>"))
	}))
	defer srv.Close()

	oldClient, oldNoRedirect := Client, ClientNoRedirect
	Client, ClientNoRedirect = srv.Client(), srv.Client()
	defer func() {
		Client, ClientNoRedirect = oldClient, oldNoRedirect
	}()

	res := FingerScan(srv.URL, http.MethodGet, true)
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Status)
	}
	if res.Location != "" {
		t.Fatalf("location = %q, want empty after follow", res.Location)
	}
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
}

// mustTestServer creates httptest.Server, skipping test if binding is not permitted in current sandbox.
func mustTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	srv, err := tryNewServer(h)
	if err != nil || srv == nil {
		t.Skipf("skip: cannot start test server (%v)", err)
	}
	return srv
}

func tryNewServer(h http.Handler) (srv *httptest.Server, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("httptest server panic: %v", r)
		}
	}()
	srv = httptest.NewServer(h)
	return srv, err
}
