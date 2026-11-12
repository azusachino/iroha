package imagefetch

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)

func allowAll(netip.AddrPort) bool { return true }

func TestPublicAddress(t *testing.T) {
	cases := map[string]bool{
		"8.8.8.8":              true,
		"1.1.1.1":              true,
		"2606:4700:4700::1111": true,
		"127.0.0.1":            false,
		"::1":                  false,
		"10.0.0.1":             false,
		"172.16.5.4":           false,
		"192.168.1.1":          false,
		"169.254.169.254":      false,
		"fe80::1":              false,
		"fc00::1":              false,
		"0.0.0.0":              false,
		"::":                   false,
		"224.0.0.1":            false,
		"255.255.255.255":      false,
		"100.64.0.1":           false,
		"240.0.0.1":            false,
		"::ffff:127.0.0.1":     false,
		"::ffff:10.0.0.1":      false,
		"::ffff:8.8.8.8":       true,
		"2001:db8::1":          false,
	}
	for in, want := range cases {
		if got := PublicAddress(netip.MustParseAddr(in)); got != want {
			t.Errorf("PublicAddress(%s) = %t, want %t", in, got, want)
		}
	}
}

func TestFetchRefusesNonPublicAddressByDefault(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()
	f := New(t.TempDir())
	if _, err := f.Fetch(context.Background(), srv.URL); !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
	if hits.Load() != 0 {
		t.Fatal("loopback server was contacted")
	}
}

func TestFetchRefusesRedirectToNonPublicAddress(t *testing.T) {
	var hits atomic.Int32
	private := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(pngBytes)
	}))
	defer private.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, private.URL, http.StatusFound)
	}))
	defer redirector.Close()
	policy := func(ap netip.AddrPort) bool { return ap.String() == strings.TrimPrefix(redirector.URL, "http://") }
	f := New(t.TempDir(), WithDialPolicy(policy))
	if _, err := f.Fetch(context.Background(), redirector.URL); !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
	if hits.Load() != 0 {
		t.Fatal("redirect target was contacted")
	}
}

func TestFetchRejectsHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png") // lying upstream header
		_, _ = w.Write([]byte("<html><script>alert(1)</script></html>"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	if _, err := New(dir, WithDialPolicy(allowAll)).Fetch(context.Background(), srv.URL); !errors.Is(err, ErrNotImage) {
		t.Fatalf("err = %v, want ErrNotImage", err)
	}
	assertEmpty(t, dir)
}

func TestFetchContentTypeComesFromSniff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()
	img, err := New(t.TempDir(), WithDialPolicy(allowAll)).Fetch(context.Background(), srv.URL)
	if err != nil || img.ContentType != "image/png" {
		t.Fatalf("img = %q, err = %v", img.ContentType, err)
	}
}

func TestFetchRejectsOversizedBodyAndDoesNotCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(pngBytes)
		chunk := make([]byte, 1<<20)
		for range 11 { // chunked, no Content-Length
			_, _ = w.Write(chunk)
			w.(http.Flusher).Flush()
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	if _, err := New(dir, WithDialPolicy(allowAll)).Fetch(context.Background(), srv.URL); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
	assertEmpty(t, dir)
}

func TestFetchCachesByURL(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()
	f := New(t.TempDir(), WithDialPolicy(allowAll))
	ctx := context.Background()
	for _, u := range []string{srv.URL + "/a", srv.URL + "/a", srv.URL + "/b"} {
		if _, err := f.Fetch(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	if hits.Load() != 2 {
		t.Fatalf("hits = %d, want 2 (one per distinct URL)", hits.Load())
	}
}

func TestFetchSharesConcurrentFetches(t *testing.T) {
	var hits atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		<-release
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()
	f := New(t.TempDir(), WithDialPolicy(allowAll))
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.Fetch(context.Background(), srv.URL); err != nil {
				t.Error(err)
			}
		}()
	}
	for hits.Load() == 0 {
	}
	close(release)
	wg.Wait()
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want 1", hits.Load())
	}
}

func TestFetchReportsCacheWriteErrorAndStillServes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	img, err := New(filepath.Join(blocker, "covers"), WithDialPolicy(allowAll)).Fetch(context.Background(), srv.URL)
	if err != nil || img.CacheErr == nil || len(img.Data) == 0 {
		t.Fatalf("img = %+v, err = %v", img, err)
	}
}

func TestFetchRejectsNonHTTPScheme(t *testing.T) {
	if _, err := New(t.TempDir()).Fetch(context.Background(), "file:///etc/passwd"); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("err = %v, want ErrInvalidURL", err)
	}
}

func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("cache dir has %d entries, want 0", len(entries))
	}
}
