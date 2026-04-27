package pictureurl

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// tinyPNG returns a 4×4 solid-color PNG as bytes.
func tinyPNG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// drainCmd executes a tea.Cmd repeatedly, feeding any imageLoadedMsg back into
// the model, until no further Cmd is produced or the message is not one we
// route. Returns the final tea.Msg observed (may be nil).
func drainCmd(t *testing.T, m *Model, cmd tea.Cmd) tea.Msg {
	t.Helper()
	var last tea.Msg
	for cmd != nil {
		msg := cmd()
		last = msg
		if msg == nil {
			return nil
		}
		cmd = m.Update(msg)
	}
	return last
}

// hasPrefix reports whether s starts with prefix (avoids panics on short s).
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestSetURL_FetchesAndDisplays(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := New()
	m.SetSize(20, 10)

	cmd := m.SetURL(srv.URL)
	if cmd == nil {
		t.Fatal("SetURL on a fresh URL should return a non-nil Cmd")
	}

	// Before the fetch resolves, View should show "Loading…".
	if got := m.View().Content; got != "Loading…" {
		t.Fatalf("expected View() == \"Loading…\" before fetch resolves, got %q", got)
	}

	drainCmd(t, &m, cmd)

	if got := m.View().Content; got == "" || got == "Loading…" {
		t.Fatalf("expected non-empty image View() after fetch, got %q", got)
	}
}

func TestSetURL_RejectsBodyOverMaxSize(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := NewWithConfig(Config{MaxSize: int64(len(body) - 1)})
	m.SetSize(20, 10)
	drainCmd(t, &m, m.SetURL(srv.URL))

	if !errors.Is(m.Err(), ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge, got %v", m.Err())
	}
	if got := m.State(); got != StateError {
		t.Fatalf("expected StateError, got %s", got)
	}
	if _, ok := m.cache[srv.URL]; ok {
		t.Fatal("oversized response should not be cached")
	}
}

func TestSetURL_RejectsDecodedPixelsOverLimit(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := NewWithConfig(Config{MaxPixels: 15})
	m.SetSize(20, 10)
	drainCmd(t, &m, m.SetURL(srv.URL))

	if !errors.Is(m.Err(), ErrImageDimensionsTooLarge) {
		t.Fatalf("expected ErrImageDimensionsTooLarge, got %v", m.Err())
	}
	if _, ok := m.cache[srv.URL]; ok {
		t.Fatal("image over pixel limit should not be cached")
	}
}

func TestSetURL_CachedErrorDoesNotRefetch(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := New()
	m.SetSize(20, 10)

	// First call: fetches, errors.
	drainCmd(t, &m, m.SetURL(srv.URL))
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 fetch after first SetURL, got %d", got)
	}
	if got := m.View().Content; !hasPrefix(got, "Image error:") {
		t.Fatalf("expected View() to start with \"Image error:\", got %q", got)
	}

	// Move away and back: must not refetch.
	m.SetURL("")
	cmd := m.SetURL(srv.URL)
	drainCmd(t, &m, cmd)
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected still 1 fetch after revisit (cached error), got %d", got)
	}
}

func TestReload_ClearsErrorAndRefetches(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	body := tinyPNG(t, color.RGBA{R: 0, G: 200, B: 0, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := New()
	m.SetSize(20, 10)

	drainCmd(t, &m, m.SetURL(srv.URL))
	if got := m.View().Content; !hasPrefix(got, "Image error:") {
		t.Fatalf("precondition: expected error View, got %q", got)
	}

	fail.Store(false)
	drainCmd(t, &m, m.Reload())
	got := m.View().Content
	if got == "" || hasPrefix(got, "Image error:") || got == "Loading…" {
		t.Fatalf("after Reload + success, expected image View, got %q", got)
	}
}

func TestDefaultCacheLimit(t *testing.T) {
	m := New()
	if m.cacheLimit != DefaultCacheLimit {
		t.Fatalf("expected default cache limit %d, got %d", DefaultCacheLimit, m.cacheLimit)
	}
}

func TestCacheLimit_EvictsOldImages(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 0, G: 0, B: 200, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	url1 := srv.URL + "/one"
	url2 := srv.URL + "/two"
	url3 := srv.URL + "/three"
	m := NewWithConfig(Config{CacheLimit: 2})
	m.SetSize(20, 10)

	drainCmd(t, &m, m.SetURL(url1))
	drainCmd(t, &m, m.SetURL(url2))
	drainCmd(t, &m, m.SetURL(url3))

	if _, ok := m.cache[url1]; ok {
		t.Fatal("oldest cached image should be evicted")
	}
	if _, ok := m.cache[url2]; !ok {
		t.Fatal("second image should remain cached")
	}
	if _, ok := m.cache[url3]; !ok {
		t.Fatal("current image should remain cached")
	}
	if got := len(m.cache); got != 2 {
		t.Fatalf("cache size = %d, want 2", got)
	}
}

func TestCacheLimit_LRUBumping(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 0, G: 0, B: 200, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	url1 := srv.URL + "/one"
	url2 := srv.URL + "/two"
	url3 := srv.URL + "/three"
	m := NewWithConfig(Config{CacheLimit: 2})
	m.SetSize(20, 10)

	// Load 1 and 2. Cache: [1, 2]
	drainCmd(t, &m, m.SetURL(url1))
	drainCmd(t, &m, m.SetURL(url2))

	// Re-access 1. Cache: [2, 1] (1 is now MRU)
	m.SetURL("") // Clear current to allow re-setting same URL
	m.SetURL(url1)

	// Load 3. Cache: [1, 3] (2 should be evicted)
	drainCmd(t, &m, m.SetURL(url3))

	if _, ok := m.cache[url2]; ok {
		t.Fatal("image 2 should have been evicted (it was the LRU after bumping 1)")
	}
	if _, ok := m.cache[url1]; !ok {
		t.Fatal("image 1 should have remained in cache due to access bump")
	}
	if _, ok := m.cache[url3]; !ok {
		t.Fatal("image 3 should remain in cache")
	}
}

func TestClear_BlanksWithoutDroppingState(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 0, G: 0, B: 200, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := New()
	m.SetSize(20, 10)
	drainCmd(t, &m, m.SetURL(srv.URL))
	if m.View().Content == "" {
		t.Fatal("precondition: expected non-empty View after fetch")
	}

	m.Clear()

	if got := m.View().Content; got != "" {
		t.Fatalf("expected empty View() after Clear(), got %q", got)
	}
	if got := m.CurrentURL(); got != srv.URL {
		t.Fatalf("Clear should not drop CurrentURL, got %q want %q", got, srv.URL)
	}
	if _, ok := m.cache[srv.URL]; !ok {
		t.Fatal("Clear should not drop cache entries")
	}
}

func TestReload_KeepsImageVisibleDuringRefetch(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 100, G: 100, B: 100, A: 255})

	// Buffered channel as a per-request gate. Pre-load one token so the first
	// request flows through immediately; the second (Reload) request will
	// block until we send another token.
	gate := make(chan struct{}, 1)
	gate <- struct{}{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-gate
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := New()
	m.SetSize(20, 10)

	// Initial fetch — the pre-loaded token lets it through.
	drainCmd(t, &m, m.SetURL(srv.URL))
	cached := m.View().Content
	if cached == "" {
		t.Fatal("precondition: expected non-empty View after initial fetch")
	}

	// Reload — gate is empty, the fetch will block until we send a token.
	reloadCmd := m.Reload()
	if reloadCmd == nil {
		t.Fatal("Reload should return a fetch Cmd")
	}

	done := make(chan tea.Msg, 1)
	go func() {
		done <- reloadCmd() // blocks until gate gets a token
	}()

	// While the refetch is in-flight, View() must still show the cached image.
	time.Sleep(20 * time.Millisecond)
	if got := m.View().Content; got != cached {
		t.Fatalf("during Reload, expected View() to keep showing cached image, got %q (was %q)", got, cached)
	}

	// Release the fetch and let it complete; feed the result through Update.
	gate <- struct{}{}
	msg := <-done
	if c := m.Update(msg); c != nil {
		_ = c() // discard any follow-up Cmd from the picture base
	}
}
