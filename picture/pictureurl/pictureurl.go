// Package pictureurl is a URL-driven layer on top of picture.Model. It owns
// HTTP fetching, per-URL image and error caches, and loading-state UI; the
// embedded picture.Model handles all rendering.
package pictureurl

import (
	"image"
	"image/color"
	"net/http"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

const (
	// DefaultMaxSize is the maximum allowed size of an image response in bytes (15 MiB).
	DefaultMaxSize = 15 * 1024 * 1024
	// DefaultTimeout is the default HTTP client timeout (15 seconds).
	DefaultTimeout = 15 * time.Second
	// DefaultMaxPixels is the maximum allowed dimensions of a decoded image (32 megapixels).
	DefaultMaxPixels = 32 * 1024 * 1024
	// DefaultCacheLimit is the default number of images to keep in the LRU cache.
	DefaultCacheLimit = 10
)

// Config configures a Model at construction.
type Config struct {
	// Base passthrough.
	KittyID    int
	Background color.Color

	// URL-specific.
	MaxSize    int64         // default 15 MiB
	MaxPixels  int           // default 32 megapixels; 0 means default, negative disables
	Timeout    time.Duration // default 15s; ignored if HTTPClient is set
	HTTPClient *http.Client  // optional; caller owns the lifetime
	CacheLimit int           // default 10; negative means unlimited
}

// Model wraps a picture.Model with URL-based fetching. Forward every tea.Msg
// to its Update; it routes fetch-completion messages internally and delegates
// everything else to the embedded picture.Model.
type Model struct {
	pic        picture.Model
	currentURL string
	cache      map[string]image.Image
	cacheOrder []string
	errs       map[string]error
	loading    map[string]bool
	maxSize    int64
	maxPixels  int
	cacheLimit int
	client     *http.Client
}

// New returns a Model with default Config.
func New() Model {
	return NewWithConfig(Config{})
}

// NewWithConfig returns a Model with the supplied Config. Zero/nil fields are
// filled with defaults.
func NewWithConfig(cfg Config) Model {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = DefaultMaxSize
	}
	if cfg.MaxPixels == 0 {
		cfg.MaxPixels = DefaultMaxPixels
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.CacheLimit == 0 {
		cfg.CacheLimit = DefaultCacheLimit
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}

	return Model{
		pic: picture.NewWithConfig(picture.Config{
			KittyID:    cfg.KittyID,
			Background: cfg.Background,
		}),
		cache:      make(map[string]image.Image),
		errs:       make(map[string]error),
		loading:    make(map[string]bool),
		maxSize:    cfg.MaxSize,
		maxPixels:  cfg.MaxPixels,
		cacheLimit: cfg.CacheLimit,
		client:     client,
	}
}

// CurrentURL returns the URL most recently passed to SetURL.
func (m *Model) CurrentURL() string { return m.currentURL }

// LoadState describes the state of the fetch for CurrentURL.
type LoadState int

const (
	// StateEmpty indicates no URL has been set.
	StateEmpty LoadState = iota
	// StateLoading indicates a fetch for the current URL is in flight.
	StateLoading
	// StateLoaded indicates the image for the current URL is cached and ready.
	StateLoaded
	// StateError indicates the fetch for the current URL failed.
	StateError
)

func (s LoadState) String() string {
	switch s {
	case StateEmpty:
		return "empty"
	case StateLoading:
		return "loading"
	case StateLoaded:
		return "loaded"
	case StateError:
		return "error"
	}
	return "unknown"
}

// State reports the load state for CurrentURL.
func (m *Model) State() LoadState {
	if m.currentURL == "" {
		return StateEmpty
	}
	if _, ok := m.errs[m.currentURL]; ok {
		return StateError
	}
	if _, ok := m.cache[m.currentURL]; ok {
		return StateLoaded
	}
	if m.loading[m.currentURL] {
		return StateLoading
	}
	return StateEmpty
}

// Err returns the cached fetch error for CurrentURL, or nil if there isn't one.
func (m *Model) Err() error {
	if m.currentURL == "" {
		return nil
	}
	return m.errs[m.currentURL]
}

// SetURL points the Model at url. If url has been fetched successfully before,
// the cached image is reused. If url is the empty string, the display clears.
// If url has previously errored, no fetch is started; call Reload to retry.
func (m *Model) SetURL(url string) tea.Cmd {
	if url == m.currentURL {
		return nil
	}
	m.currentURL = url

	if url == "" {
		return m.pic.SetImage(nil)
	}

	if img, ok := m.cache[url]; ok {
		return m.pic.SetImage(img)
	}
	if _, ok := m.errs[url]; ok {
		return m.pic.SetImage(nil)
	}
	if m.loading[url] {
		return m.pic.SetImage(nil)
	}

	m.loading[url] = true
	clearCmd := m.pic.SetImage(nil)
	return tea.Batch(clearCmd, fetchCmd(m.client, url, m.maxSize, m.maxPixels))
}

// Reload re-fetches CurrentURL. The currently-displayed image (if any) stays
// visible during the refetch; on success it is replaced by the new image, on
// failure the display blanks and View() shows the error.
func (m *Model) Reload() tea.Cmd {
	if m.currentURL == "" {
		return nil
	}
	delete(m.errs, m.currentURL)
	m.loading[m.currentURL] = true
	return fetchCmd(m.client, m.currentURL, m.maxSize, m.maxPixels)
}

// Clear blanks the display without dropping CurrentURL or any cache entries.
// Useful for combining with Reload to get blank-during-refetch.
func (m *Model) Clear() tea.Cmd {
	return m.pic.SetImage(nil)
}

// SetSize forwards to the embedded picture.Model.
func (m *Model) SetSize(cols, rows int) tea.Cmd { return m.pic.SetSize(cols, rows) }

// Toggle forwards to the embedded picture.Model.
func (m *Model) Toggle() tea.Cmd { return m.pic.Toggle() }

// Mode forwards to the embedded picture.Model.
func (m *Model) Mode() picture.PictureMode { return m.pic.Mode() }

// Update routes fetch-completion messages and delegates everything else to
// the embedded picture.Model.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if loaded, ok := msg.(ImageLoadedMsg); ok {
		delete(m.loading, loaded.URL)
		if loaded.Err != nil {
			m.errs[loaded.URL] = loaded.Err
			delete(m.cache, loaded.URL)
			if loaded.URL == m.currentURL {
				return m.pic.SetImage(nil)
			}
			return nil
		}
		m.rememberImage(loaded.URL, loaded.Img)
		delete(m.errs, loaded.URL)
		if loaded.URL == m.currentURL {
			return m.pic.SetImage(loaded.Img)
		}
		return nil
	}
	return m.pic.Update(msg)
}

// View returns the appropriate placeholder text for URL-layer states (empty,
// error, loading) and otherwise delegates to the embedded picture.Model.
func (m *Model) View() tea.View {
	if m.currentURL == "" {
		return tea.NewView("")
	}
	if _, hasImg := m.cache[m.currentURL]; !hasImg {
		if err, errored := m.errs[m.currentURL]; errored {
			return tea.NewView("Image error:\n" + err.Error())
		}
		if m.loading[m.currentURL] {
			return tea.NewView("Loading…")
		}
	}
	return m.pic.View()
}

// String returns the rendered image content as a plain string.
func (m *Model) String() string { return m.View().Content }

// IsPictureMsg reports whether msg is an async update owned by the pictureurl
// layer (image fetch completions) or the embedded picture.Model (Kitty frames).
func IsPictureMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case ImageLoadedMsg:
		return true
	}
	return picture.IsPictureMsg(msg)
}

func (m *Model) rememberImage(url string, img image.Image) {
	if _, ok := m.cache[url]; !ok {
		m.cacheOrder = append(m.cacheOrder, url)
	}
	m.cache[url] = img
	m.trimCache()
}

func (m *Model) trimCache() {
	if m.cacheLimit < 0 {
		return
	}
	for len(m.cacheOrder) > m.cacheLimit {
		evict := m.cacheOrder[0]
		m.cacheOrder = m.cacheOrder[1:]
		if evict == m.currentURL {
			m.cacheOrder = append(m.cacheOrder, evict)
			continue
		}
		delete(m.cache, evict)
	}
}
