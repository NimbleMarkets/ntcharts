// Package heatpicture renders a continuous 2D scalar field as a high-res
// heatmap via the Kitty graphics protocol (with a half-block-glyph fallback
// in Glyph mode). The data source is a Sampler — a function from data
// coordinates to a scalar value. Each frame samples the sampler at full
// terminal-pixel resolution (cellPixelW × cols  by  cellPixelH × rows
// pixels), maps each sample through the configured color scale with linear
// interpolation between scale stops, and feeds the resulting image.Image to
// an embedded picture.Model for encoding and placement.
//
// Compared to the cell-grid heatmap.Model, heatpicture trades ANSI
// universality for a visually smooth gradient field (true per-pixel
// resolution under Kitty, no cell-quantization artifacts) and is the right
// fit for function-driven demos like Perlin noise.
package heatpicture

import (
	"fmt"
	"image"
	"image/color"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// DefaultKittyID is offset from picture.DefaultKittyID so heatpicture and
// picture.Model can coexist in the same program with default configs.
const DefaultKittyID = picture.DefaultKittyID + 100

// Sampler maps a data-space (x, y) to a scalar. Called once per source
// pixel during rendering, so a cheap implementation is preferable for
// animated demos. The renderer normalizes the returned value to [0,1] via
// (val-minVal)/(maxVal-minVal) before sampling the color scale.
type Sampler func(x, y float64) float64

// Config configures a Model at construction. Zero/nil fields take the
// embedded picture.Config defaults plus heatpicture's own defaults
// (DefaultKittyID, no sampler, no scale, [0,1] value range, [0,1]×[0,1]
// data range, SamplingFactor 1.0).
type Config struct {
	KittyID         int
	Background      color.Color
	CellPixelWidth  int
	CellPixelHeight int

	// SamplingFactor scales the source pixel resolution in Kitty mode.
	// 1.0 (default) samples at full terminal-pixel resolution
	// (cols*cellPixelW × rows*cellPixelH) for crisp 1:1 placement.
	// Values <1 reduce sampling resolution by that fraction, letting
	// Kitty interpolate during placement — useful for large terminals
	// where per-frame sampling cost would otherwise stall animation.
	// 0.5 = quarter the work; 0.25 = sixteenth. Glyph mode is unaffected
	// (already at half-block resolution). Out-of-range values are
	// clamped to (0, 1].
	SamplingFactor float64

	// Fit controls how the rendered field is mapped onto the cell
	// rectangle in the embedded picture.Model. The zero value is
	// FitContain. In practice heatpicture renders at exactly the cell-rect
	// pixel dimensions, so all three modes look identical here — the
	// field exists for API parity with picture.Model.
	Fit picture.FitMode
}

// Model wraps picture.Model with a continuous heatmap as the image source.
// Standard tea-component surface (Init / Update / View / SetSize / Toggle).
// Setters return tea.Cmd that, when there's enough state to render
// (sampler set, size set, scale set), schedule an async sampling job.
type Model struct {
	pic picture.Model

	sampler    Sampler
	colorScale []color.Color
	rgbaScale  []color.RGBA // precomputed from colorScale; passed to sampleField

	minVal, maxVal         float64
	minX, maxX, minY, maxY float64

	cols, rows     int
	cellW, cellH   int
	samplingFactor float64

	modelID uint64
	seq     uint64
	err     error

	// renderPending is true while a sampling Cmd is outstanding. dirty is
	// set when a parameter change came in during that window so the
	// completion handler can kick a follow-up render. Together they
	// throttle bursty setter traffic (e.g. an animation tick that fires
	// faster than a frame can render at large terminal sizes) to "one
	// render in flight at a time".
	renderPending bool
	dirty         bool

	// lastRenderDuration is the wall-time spent in the most recent
	// render closure (sampler + colorAt + Pix writes). Updated on every
	// heatRenderedMsg arrival so consumers can show a per-frame cost
	// meter while seq is churning under bursty setters.
	lastRenderDuration time.Duration

	// frameCount counts how many own-modelID heatRenderedMsg arrivals
	// have been processed (i.e., renders that actually completed).
	// compositeCount counts how many of those resulted in pic.SetImage
	// being called with a non-nil image (i.e., were actually pushed to
	// the picture layer for display). Diagnostic for "renders are
	// happening but not getting composited" — the two should track
	// closely; a divergence implies errors or cross-Model rejection.
	frameCount     uint64
	compositeCount uint64
}

var nextModelID atomic.Uint64

// New returns a Model with default Config.
func New() Model {
	return NewWithConfig(Config{})
}

// NewWithConfig returns a Model with the supplied Config. Zero/nil fields
// get heatpicture defaults; the embedded picture.Model takes the cell
// pixel size so its APC encodes at the right display dims.
func NewWithConfig(cfg Config) Model {
	if cfg.KittyID <= 0 {
		cfg.KittyID = DefaultKittyID
	}
	if cfg.CellPixelWidth <= 0 {
		cfg.CellPixelWidth = 8
	}
	if cfg.CellPixelHeight <= 0 {
		cfg.CellPixelHeight = 16
	}
	if cfg.SamplingFactor <= 0 || cfg.SamplingFactor > 1 {
		cfg.SamplingFactor = 1.0
	}
	picCfg := picture.Config{
		KittyID:               cfg.KittyID,
		Background:            cfg.Background,
		Fit:                   cfg.Fit,
		CellPixelWidth:        cfg.CellPixelWidth,
		CellPixelHeight:       cfg.CellPixelHeight,
		KittyResolutionFactor: cfg.SamplingFactor,
	}
	return Model{
		pic:     picture.NewWithConfig(picCfg),
		modelID: nextModelID.Add(1),
		// Default-friendly ranges so a demo that only sets a sampler still
		// renders something sensible.
		minVal: 0, maxVal: 1,
		minX: 0, maxX: 1, minY: 0, maxY: 1,
		cellW:          cfg.CellPixelWidth,
		cellH:          cfg.CellPixelHeight,
		samplingFactor: cfg.SamplingFactor,
	}
}

// SamplingFactor returns the current sampling-factor multiplier (see
// Config.SamplingFactor).
func (m *Model) SamplingFactor() float64 { return m.samplingFactor }

// LastRenderDuration returns the wall-time spent inside the most recent
// render closure (sampler + colorAt + Pix writes). Useful for surfacing
// a per-frame cost meter — large values at large terminal sizes are the
// signal to lower SamplingFactor or slow the animation tick.
func (m *Model) LastRenderDuration() time.Duration { return m.lastRenderDuration }

// FrameCount returns the number of own-modelID heatRenderedMsg arrivals
// processed. Increments once per render that completed and reached
// Update — pairs with CompositeCount as a "renders happening?" vs.
// "frames making it through to the picture layer?" diagnostic.
func (m *Model) FrameCount() uint64 { return m.frameCount }

// CompositeCount returns the number of frames pushed to the embedded
// picture.Model via SetImage with a non-nil image. Should track
// FrameCount closely; a sustained gap means renders are being rejected
// (cross-talk) or producing errors.
func (m *Model) CompositeCount() uint64 { return m.compositeCount }

// CellPixelSize returns the terminal cell pixel size (initial 8×16 default
// or whatever the terminal reported via uv.CellSizeEvent). Used together
// with SamplePixelSize to confirm sampling resolution.
func (m *Model) CellPixelSize() (w, h int) { return m.cellW, m.cellH }

// SamplePixelSize returns the source-image pixel dimensions the next
// renderCmd would produce: cellW×cols × cellH×rows × samplingFactor in
// Kitty mode, cols × rows*2 in Glyph mode. Diagnostic for "is the source
// actually being sampled at the resolution I expect".
func (m *Model) SamplePixelSize() (w, h int) {
	if m.pic.Mode() == picture.PictureKitty {
		w = int(float64(m.cols*m.cellW) * m.samplingFactor)
		h = int(float64(m.rows*m.cellH) * m.samplingFactor)
	} else {
		w = m.cols
		h = m.rows * 2
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// SetSamplingFactor updates the sampling-factor multiplier at runtime
// (see Config.SamplingFactor). Out-of-range values are clamped to (0, 1].
// Also propagates to the embedded picture.Model's KittyResolutionFactor
// so the transmitted Kitty image shrinks in lockstep with the source
// bitmap — one knob controls both Perlin compute cost and displayed
// Kitty fidelity. Returns a render Cmd if the new value differs and a
// render isn't already in flight; nil otherwise.
func (m *Model) SetSamplingFactor(f float64) tea.Cmd {
	if f <= 0 || f > 1 {
		f = 1.0
	}
	if f == m.samplingFactor {
		return nil
	}
	m.samplingFactor = f
	m.seq++
	// Propagate to picture's factor for state consistency. Discard its
	// returned render Cmd — heatpicture's scheduleRender (below) will
	// produce a fresh image whose SetImage triggers a picture render at
	// the new factor anyway. Avoids two redundant Kitty re-encodes.
	_ = m.pic.SetKittyResolutionFactor(f)
	return m.scheduleRender()
}

// SetSampler replaces the sampling function and schedules a fresh render
// (or returns nil if size or scale aren't set yet, or a render is already
// in flight — in which case the change is queued via the dirty flag).
func (m *Model) SetSampler(f Sampler) tea.Cmd {
	m.sampler = f
	m.seq++
	m.err = nil
	return m.scheduleRender()
}

// SetXYRange sets the data-space rectangle that maps onto the rendered
// pixel grid. minX maps to pixel column 0; maxX to the rightmost column.
// Returns a render Cmd if state is sufficient and no render is in flight.
func (m *Model) SetXYRange(minX, maxX, minY, maxY float64) tea.Cmd {
	m.minX, m.maxX, m.minY, m.maxY = minX, maxX, minY, maxY
	m.seq++
	return m.scheduleRender()
}

// SetValueRange sets the value clamp used to normalize sampler output to
// [0,1] for the color scale. Values outside [min,max] are clamped to the
// boundary. Returns a render Cmd if state is sufficient and no render is
// in flight.
func (m *Model) SetValueRange(min, max float64) tea.Cmd {
	m.minVal, m.maxVal = min, max
	m.seq++
	return m.scheduleRender()
}

// SetColorScale sets the gradient scale. Linear interpolation between
// adjacent stops produces sub-cell smoothness — that's the visual win
// over heatmap.Model. Returns a render Cmd if state is sufficient and no
// render is in flight.
func (m *Model) SetColorScale(scale []color.Color) tea.Cmd {
	m.colorScale = scale
	m.rgbaScale = toRGBA(scale)
	m.seq++
	return m.scheduleRender()
}

// scheduleRender returns a fresh render Cmd if no render is currently
// in flight, or nil with the dirty flag set so the completion handler
// will kick a follow-up render when the in-flight one finishes. This is
// the throttle: setters never queue more than one render's worth of work
// behind the goroutine pool.
func (m *Model) scheduleRender() tea.Cmd {
	if m.renderPending {
		m.dirty = true
		return nil
	}
	return m.startRender()
}

// startRender unconditionally dispatches a render Cmd if state is
// sufficient. Sets renderPending. Used both for fresh schedules and for
// the dirty-followup after a render completes.
func (m *Model) startRender() tea.Cmd {
	cmd := m.renderCmd()
	if cmd != nil {
		m.renderPending = true
		m.dirty = false
	}
	return cmd
}

// Init forwards to the embedded picture.Model so the terminal's real cell
// pixel size is queried at startup; the response (uv.CellSizeEvent) is
// intercepted by Update and triggers a fresh render at the new resolution.
func (m *Model) Init() tea.Cmd { return m.pic.Init() }

// SetSize updates the rendering dimensions in terminal cells. Forwards to
// the embedded picture.Model, then re-renders the sampler at the new
// pixel dims (subject to the throttle: if a render is in flight, the
// follow-up will pick up the new size).
func (m *Model) SetSize(cols, rows int) tea.Cmd {
	if cols < 0 {
		cols = 0
	}
	if rows < 0 {
		rows = 0
	}
	if cols == m.cols && rows == m.rows {
		return nil
	}
	m.cols = cols
	m.rows = rows
	m.seq++
	picCmd := m.pic.SetSize(cols, rows)
	heatCmd := m.scheduleRender()
	if picCmd == nil {
		return heatCmd
	}
	if heatCmd == nil {
		return picCmd
	}
	return tea.Batch(picCmd, heatCmd)
}

// Toggle switches between Glyph and Kitty modes. Forwards to picture.Model
// and also re-samples at the new mode's preferred resolution (Kitty wants
// full terminal-pixel resolution; Glyph only needs one pixel per
// half-block, which is much faster at large sizes). On Glyph→Kitty the
// re-encode-old-image Cmd from picture.Toggle is discarded — heatpicture
// will push a fresh image at the new resolution, so the upscale of the
// stale Glyph-res bitmap would just flicker briefly before being
// overwritten. On Kitty→Glyph the picture-layer Cmd may be a
// kittyDeleteImage cleanup, which is preserved.
func (m *Model) Toggle() tea.Cmd {
	prevMode := m.pic.Mode()
	picCmd := m.pic.Toggle()
	m.seq++
	heatCmd := m.scheduleRender()
	if prevMode == picture.PictureGlyph {
		// Glyph→Kitty: drop picture's re-encode-old Cmd.
		return heatCmd
	}
	// Kitty→Glyph: keep picCmd (cleanup).
	if picCmd == nil {
		return heatCmd
	}
	if heatCmd == nil {
		return picCmd
	}
	return tea.Batch(picCmd, heatCmd)
}

// Mode returns the current rendering mode.
func (m *Model) Mode() picture.PictureMode { return m.pic.Mode() }

// Fit forwards to the embedded picture.Model.
func (m *Model) Fit() picture.FitMode { return m.pic.Fit() }

// SetFit forwards to the embedded picture.Model.
func (m *Model) SetFit(fit picture.FitMode) tea.Cmd { return m.pic.SetFit(fit) }

// KittySupported forwards to the embedded picture.Model — Kitty
// capability is process-wide.
func (m *Model) KittySupported() picture.KittyCapability { return m.pic.KittySupported() }

// Err returns the last sampling/encoding error, or nil if the most recent
// render succeeded.
func (m *Model) Err() error { return m.err }

// String returns the rendered content as a plain string.
func (m *Model) String() string { return m.View().Content }

// Update routes heatpicture-owned async messages and delegates everything
// else to the embedded picture.Model. Intercepts uv.CellSizeEvent so a
// real-cell-size update from the terminal triggers a fresh sample at the
// new pixel resolution rather than just a rescale of the previous bitmap.
//
// On heatRenderedMsg arrival the throttle is cleared (renderPending=false);
// if a parameter changed during the render (dirty=true), a follow-up
// render is dispatched. Stale frames (seq mismatch from cross-talk or
// from a setter bumping seq mid-flight) are dropped on the floor for
// SetImage but still clear the pending flag.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case heatRenderedMsg:
		// Cross-talk between Models: only accept frames produced by our
		// own renderCmd (one Cmd per Model, atomic modelID counter).
		if msg.modelID != m.modelID {
			return nil
		}
		// Always accept our own in-flight render. The throttle ensures
		// only one Cmd is in flight at a time, so msg.seq < m.seq just
		// means parameters changed during the render — the rendered
		// image is stale relative to the latest state but it's still
		// the freshest thing we have. Rejecting it would leave the
		// previous frame onscreen forever when the per-frame cost
		// exceeds the inter-tick spacing (steady seq churn).
		m.lastRenderDuration = msg.duration
		m.frameCount++
		m.renderPending = false
		var cmd tea.Cmd
		if msg.err != nil {
			m.err = msg.err
			cmd = m.pic.SetImage(nil)
		} else {
			m.err = nil
			cmd = m.pic.SetImage(msg.img)
			m.compositeCount++
		}
		// Parameter changed during render → pick up the latest state.
		if m.dirty {
			followUp := m.startRender()
			if cmd != nil && followUp != nil {
				return tea.Batch(cmd, followUp)
			}
			if followUp != nil {
				return followUp
			}
		}
		return cmd

	case uv.CellSizeEvent:
		// Update local cell pixel size and the embedded picture.Model's,
		// then trigger a fresh sample at the new pixel resolution. The
		// re-encode Cmd that picture.SetCellPixelSize would normally
		// return is intentionally discarded — we're about to push a new
		// image at the new dims, so re-encoding the old one is wasted.
		w, h := msg.Width, msg.Height
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		m.cellW, m.cellH = w, h
		_ = m.pic.SetCellPixelSize(w, h)
		m.seq++
		return m.scheduleRender()
	}
	return m.pic.Update(msg)
}

// View returns the rendered content, or an error placeholder if the most
// recent render failed.
func (m *Model) View() tea.View {
	if m.err != nil {
		return tea.NewView("Heatmap error:\n" + m.err.Error())
	}
	return m.pic.View()
}

// renderCmd schedules an asynchronous sampling job at a resolution chosen
// for the current rendering mode:
//
//   - Glyph mode: sample at (cols, rows*2) — exactly one pixel per
//     half-block, since ansimage will downscale anything larger to that
//     anyway. Massive speedup at large terminal sizes (a 200×60 cell
//     terminal samples 24K pixels instead of 1.5M).
//   - Kitty mode: sample at (cols*cellW, rows*cellH) — full terminal
//     pixel resolution so the Kitty APC renders crisp 1:1 with no
//     downscale loss.
//
// Returns nil if any of (sampler, color scale, cols, rows) is missing —
// the caller is expected to treat that as "not enough state to render
// yet" rather than an error.
func (m *Model) renderCmd() tea.Cmd {
	if m.sampler == nil || len(m.rgbaScale) == 0 || m.cols <= 0 || m.rows <= 0 {
		return nil
	}
	var pixelW, pixelH int
	if m.pic.Mode() == picture.PictureKitty {
		pixelW = int(float64(m.cols*m.cellW) * m.samplingFactor)
		pixelH = int(float64(m.rows*m.cellH) * m.samplingFactor)
		if pixelW < 1 {
			pixelW = 1
		}
		if pixelH < 1 {
			pixelH = 1
		}
	} else {
		// Match the cell rectangle's pixel aspect ratio so prepareSource's
		// Fit step letterboxes the Glyph bitmap identically to Kitty (both
		// source ARs equal target AR → Contain/Fill/Cover all produce the
		// same visual result). The typical 1:2 cell aspect (8×16 default)
		// yields cellH/cellW = 2, preserving the original "one pixel per
		// half-block" shortcut. Non-1:2 cell aspects — e.g. ghostty-web
		// reporting 10×16 — get a slightly different pixelH but matching
		// AR so the user can't tell Glyph from Kitty by the fit bars.
		pixelW = m.cols
		pixelH = m.rows * m.cellH / m.cellW
		if pixelH < 1 {
			pixelH = 1
		}
	}
	// Snapshot all sampling parameters into the closure so a subsequent
	// setter call doesn't race against the goroutine.
	modelID, seq := m.modelID, m.seq
	sampler := m.sampler
	scale := m.rgbaScale
	minV, maxV := m.minVal, m.maxVal
	minX, maxX, minY, maxY := m.minX, m.maxX, m.minY, m.maxY
	return func() tea.Msg {
		start := time.Now()
		img, err := sampleField(sampler, scale, minV, maxV, minX, maxX, minY, maxY, pixelW, pixelH)
		return heatRenderedMsg{
			modelID:  modelID,
			seq:      seq,
			img:      img,
			err:      err,
			duration: time.Since(start),
		}
	}
}

// sampleField builds the high-res heatmap bitmap. Each pixel in the
// pixelW × pixelH output is mapped to a data-space (x, y) by linear
// interpolation across the configured (minX,maxX) × (minY,maxY) rectangle,
// passed through sampler, normalized to [0,1] via the value range, and
// looked up in the pre-converted RGBA color scale.
//
// Hot path: writes directly into the underlying Pix slice rather than
// going through img.Set or img.SetRGBA. img.Set boxes the color into a
// color.Color interface (one alloc per pixel), and img.SetRGBA still
// re-validates bounds and computes the offset; for ~1.6M pixels at large
// terminal sizes that overhead dominates the per-frame cost. Direct
// writes drop the allocs from "one per pixel" to zero.
func sampleField(
	sampler Sampler, scale []color.RGBA,
	minV, maxV, minX, maxX, minY, maxY float64,
	pixelW, pixelH int,
) (image.Image, error) {
	if pixelW <= 0 || pixelH <= 0 {
		return nil, fmt.Errorf("heatpicture: zero pixel size (%dx%d)", pixelW, pixelH)
	}
	valSpan := maxV - minV
	if valSpan == 0 {
		// Avoid div-by-zero; treat all samples as t=0.
		valSpan = 1
	}
	xSpan := maxX - minX
	ySpan := maxY - minY
	img := image.NewRGBA(image.Rect(0, 0, pixelW, pixelH))
	stride := img.Stride
	pix := img.Pix
	invValSpan := 1.0 / valSpan
	denomY := float64(pixelH - 1)
	denomX := float64(pixelW - 1)
	for py := 0; py < pixelH; py++ {
		var ty float64
		if pixelH > 1 {
			ty = float64(py) / denomY
		}
		dy := minY + ySpan*ty
		rowOff := py * stride
		for px := 0; px < pixelW; px++ {
			var tx float64
			if pixelW > 1 {
				tx = float64(px) / denomX
			}
			dx := minX + xSpan*tx
			val := sampler(dx, dy)
			t := (val - minV) * invValSpan
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			c := colorAt(scale, t)
			off := rowOff + px*4
			pix[off] = c.R
			pix[off+1] = c.G
			pix[off+2] = c.B
			pix[off+3] = c.A
		}
	}
	return img, nil
}

// IsHeatPictureMsg reports whether msg is owned by the heatpicture layer or
// the embedded picture.Model. Consumers that gate forwarding on this helper
// must include uv.CellSizeEvent in their forwarding decision because Update
// intercepts it to trigger a fresh sample.
func IsHeatPictureMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case heatRenderedMsg:
		return true
	}
	return picture.IsPictureMsg(msg)
}
