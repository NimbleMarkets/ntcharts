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

// TestInit_DispatchesCellSizeRequest verifies the pictureurl wrapper's Init
// bubbles the picture-layer cell-size request so consumers can call a single
// Init Cmd and have terminal-reported cell dims auto-applied for Kitty
// placement.
func TestInit_DispatchesCellSizeRequest(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init must return a non-nil Cmd")
	}
	if !batchContainsCSI16t(cmd) {
		t.Error("expected Init to include a CSI 16 t request (possibly batched with the Kitty support probe)")
	}
}

// batchContainsCSI16t walks a Cmd (running it, then walking a BatchMsg
// if produced) and reports whether any sub-Cmd produces a tea.RawMsg
// carrying "\x1b[16t".
func batchContainsCSI16t(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			if batchContainsCSI16t(sub) {
				return true
			}
		}
		return false
	}
	if raw, ok := msg.(tea.RawMsg); ok {
		if seq, _ := raw.Msg.(string); seq == "\x1b[16t" {
			return true
		}
	}
	return false
}

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

func TestSetURL_SendsUserAgentAndAcceptHeaders(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	var gotUA, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := New()
	m.SetSize(20, 10)
	drainCmd(t, &m, m.SetURL(srv.URL))

	if !hasPrefix(gotUA, "ntcharts-pictureurl") {
		t.Fatalf("expected User-Agent to start with %q, got %q", "ntcharts-pictureurl", gotUA)
	}
	if !hasPrefix(gotAccept, "image/") {
		t.Fatalf("expected Accept to start with %q, got %q", "image/", gotAccept)
	}
}

func TestSetURL_HonorsCustomUserAgent(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	m := NewWithConfig(Config{UserAgent: "MyApp/1.2.3"})
	m.SetSize(20, 10)
	drainCmd(t, &m, m.SetURL(srv.URL))

	if gotUA != "MyApp/1.2.3" {
		t.Fatalf("expected User-Agent %q, got %q", "MyApp/1.2.3", gotUA)
	}
}

func TestSetURL_RejectsNonImageContentType(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(body)
	}))
	defer srv.Close()

	m := New()
	m.SetSize(20, 10)
	drainCmd(t, &m, m.SetURL(srv.URL))

	if !errors.Is(m.Err(), ErrUnexpectedContentType) {
		t.Fatalf("expected ErrUnexpectedContentType, got %v", m.Err())
	}
	if got := m.State(); got != StateError {
		t.Fatalf("expected StateError, got %s", got)
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

func TestUpdate_IgnoresImageLoadedMsgFromOtherModel(t *testing.T) {
	body := tinyPNG(t, color.RGBA{R: 200, G: 0, B: 0, A: 255})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(body)
	}))
	defer srv.Close()

	tooSmall := NewWithConfig(Config{MaxSize: int64(len(body) - 1)})
	other := New()
	tooSmall.SetSize(20, 10)
	other.SetSize(20, 10)

	cmd := tooSmall.SetURL(srv.URL)
	if cmd == nil {
		t.Fatal("SetURL should return a fetch Cmd")
	}
	if otherCmd := other.SetURL(srv.URL); otherCmd == nil {
		t.Fatal("other SetURL should return a fetch Cmd")
	}

	msg := cmd()
	loaded, ok := msg.(ImageLoadedMsg)
	if !ok {
		t.Fatalf("expected ImageLoadedMsg, got %T", msg)
	}
	if loaded.Err == nil {
		t.Fatal("precondition: first model should produce an error")
	}

	if out := other.Update(loaded); out != nil {
		t.Fatalf("message from another model should be ignored, got Cmd %v", out)
	}
	if err := other.Err(); err != nil {
		t.Fatalf("other model should not receive foreign error, got %v", err)
	}
	if got := other.State(); got != StateLoading {
		t.Fatalf("other model state = %s, want loading", got)
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

func TestUpdate_StaleEarlierFetchDiscarded(t *testing.T) {
	// Simulate the race fixed by I1: two fetches for the same URL are in
	// flight (initial SetURL + Reload). The second fetch (latest) returns
	// first and updates the view. The first fetch (now stale) returns later
	// and must NOT clobber the latest result.
	const url = "http://example.com/x"
	m := New()
	m.SetSize(20, 10)

	// Drive the model into a state where two fetches have been dispatched
	// for the same URL.
	first := m.SetURL(url)
	if first == nil {
		t.Fatal("SetURL should return a fetch Cmd")
	}
	second := m.Reload()
	if second == nil {
		t.Fatal("Reload should return a fetch Cmd")
	}

	imgStale := image.NewRGBA(image.Rect(0, 0, 4, 4))
	imgFresh := image.NewRGBA(image.Rect(0, 0, 4, 4))

	// Latest fetch (seq=2) returns first: accepted.
	m.Update(ImageLoadedMsg{modelID: m.modelID, seq: m.fetchSeq[url], URL: url, Img: imgFresh})
	if got := m.cache[url]; got != imgFresh {
		t.Fatalf("after latest fetch, cache should hold imgFresh, got %p (want %p)", got, imgFresh)
	}

	// Earlier fetch (seq=1) returns later — must be discarded.
	m.Update(ImageLoadedMsg{modelID: m.modelID, seq: m.fetchSeq[url] - 1, URL: url, Img: imgStale})
	if got := m.cache[url]; got != imgFresh {
		t.Fatalf("stale fetch clobbered cache; got %p, want %p (imgFresh)", got, imgFresh)
	}
}

func TestTrimCache_BailsWhenNoProgressPossible(t *testing.T) {
	// Construct the degenerate state I5 warns about: cacheLimit=0 with
	// currentURL pinned as the sole entry. The pre-fix loop would pop and
	// re-append currentURL forever.
	m := New()
	m.cacheLimit = 0
	m.currentURL = "http://x"
	m.cache["http://x"] = image.NewRGBA(image.Rect(0, 0, 1, 1))
	m.cacheOrder = []string{"http://x"}

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.trimCache()
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("trimCache hung — expected to return when no progress can be made")
	}

	if _, ok := m.cache["http://x"]; !ok {
		t.Fatal("currentURL must not be evicted")
	}
}

func TestTrimCache_EvictsNonCurrentEvenAtZeroLimit(t *testing.T) {
	// With cacheLimit=0 and an entry that isn't currentURL, trimCache should
	// evict the non-currentURL entry and then return (not loop on currentURL).
	m := New()
	m.cacheLimit = 0
	m.currentURL = "http://current"
	m.cache["http://current"] = image.NewRGBA(image.Rect(0, 0, 1, 1))
	m.cache["http://old"] = image.NewRGBA(image.Rect(0, 0, 1, 1))
	m.cacheOrder = []string{"http://old", "http://current"}

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.trimCache()
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("trimCache hung")
	}

	if _, ok := m.cache["http://old"]; ok {
		t.Fatal("non-current entry should have been evicted")
	}
	if _, ok := m.cache["http://current"]; !ok {
		t.Fatal("currentURL must remain")
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
