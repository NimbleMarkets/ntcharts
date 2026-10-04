// ntcharts - Copyright (c) 2026 Neomantra Corp.

package timeserieslinechart

import (
	"math"
	"sort"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// FitYOpts configures fitting the Y viewport to stored points in the time viewport.
type FitYOpts struct {
	// DataSets selects the data sets to fit. Nil selects all data sets;
	// an empty, non-nil slice selects none. Unknown names are ignored.
	DataSets []string
	// IncludeZero includes a zero baseline on a linear Y axis.
	// It is ignored on a logarithmic Y axis.
	IncludeZero bool
}

// FitYToView fits the displayed Y range to all stored points whose timestamps
// are inside the current time viewport, including both endpoints.
// See FitYToViewWithOpts for dataset selection and a zero baseline.
func (m *Model) FitYToView() bool {
	return m.FitYToViewWithOpts(FitYOpts{})
}

// FitYToViewWithOpts fits the displayed Y range to the selected data sets.
// It ignores non-finite values and values invalid on the Y scale. If there
// are no eligible points, it returns false and leaves the ranges unchanged.
// Constant values get a nonzero range (5% or at least 1 on a linear axis,
// half a decade on either side on a log axis). Expected Y bounds expand if
// needed but never shrink, so retained history remains available for zooming.
// AutoMinY and AutoMaxY are unchanged; this explicit fit overrides manual
// Y zoom. Only stored points are considered, not interpolated boundary values.
func (m *Model) FitYToViewWithOpts(opts FitYOpts) bool {
	names := opts.DataSets
	if names == nil {
		names = make([]string, 0, len(m.dSets))
		for name := range m.dSets {
			names = append(names, name)
		}
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, name := range names {
		ds, exists := m.dSets[name]
		if !exists {
			continue
		}
		for _, p := range ds.tBuf.ReadAllRaw() {
			if p.X < m.ViewMinX() || p.X > m.ViewMaxX() ||
				math.IsNaN(p.Y) || math.IsInf(p.Y, 0) || !m.YScale().Valid(p.Y) {
				continue
			}
			lo, hi = min(lo, p.Y), max(hi, p.Y)
		}
	}
	if lo > hi {
		return false
	}
	if opts.IncludeZero && m.YScale() != linechart.ScaleLog {
		lo, hi = min(lo, 0), max(hi, 0)
	}
	if lo == hi {
		if m.YScale() == linechart.ScaleLog {
			lo, hi = max(math.SmallestNonzeroFloat64, lo/math.Sqrt(10)), min(math.MaxFloat64, hi*math.Sqrt(10))
		} else {
			pad := max(1, math.Abs(lo)*0.05)
			lo, hi = max(-math.MaxFloat64, lo-pad), min(math.MaxFloat64, hi+pad)
			if opts.IncludeZero && lo < 0 && hi > 0 {
				lo = 0
			}
		}
	}
	m.Model.SetYRange(min(m.MinY(), lo), max(m.MaxY(), hi))
	m.Model.SetViewYRange(lo, hi)
	m.rescaleData()
	return true
}

// TrimBefore removes points strictly before t from every data set and returns
// the number removed. Points exactly at t are retained. Like Push, timestamps
// are compared at millisecond resolution, and data must be chronological.
// Surviving points are copied to new buffers so discarded storage is released.
// Styles, expected ranges and viewports are unchanged; call FitYToView to
// refit Y afterwards. To preserve interpolation at the left edge, trim at an
// earlier timestamp than the viewport's start to keep a preceding sample.
func (m *Model) TrimBefore(t time.Time) int {
	cutoff := float64(t.UnixMilli()) / 1e3
	removed := 0
	for _, ds := range m.dSets {
		raw := ds.tBuf.ReadAllRaw()
		n := sort.Search(len(raw), func(i int) bool { return raw[i].X >= cutoff })
		if n > 0 {
			ds.tBuf.SetData(raw[n:])
			removed += n
		}
	}
	return removed
}
