// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"math"
	"strconv"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// FormatValue renders v according to f. Zero-valued fields fall back to
// per-kind defaults; an empty Kind renders a plain minimal float.
func FormatValue(f Format, v float64) string {
	prec := -1
	if f.Precision != nil {
		prec = *f.Precision
	}
	switch f.Kind {
	case "percent":
		if prec < 0 {
			prec = 0
		}
		return strconv.FormatFloat(v*100, 'f', prec, 64) + "%"
	case "currency":
		if prec < 0 {
			prec = 2
		}
		sym := f.Currency
		if sym == "" {
			sym = "$"
		}
		return sym + strconv.FormatFloat(v, 'f', prec, 64)
	case "si":
		return formatSI(v, prec)
	case "time":
		layout := f.Layout
		if layout == "" {
			layout = "2006-01-02"
		}
		return time.UnixMilli(int64(v)).UTC().Format(layout)
	default: // "" or "number"
		return strconv.FormatFloat(v, 'f', prec, 64)
	}
}

var siSuffixes = []struct {
	factor float64
	suffix string
}{
	{1e12, "T"},
	{1e9, "G"},
	{1e6, "M"},
	{1e3, "k"},
}

func formatSI(v float64, prec int) string {
	if prec < 0 {
		prec = 1
	}
	abs := math.Abs(v)
	for _, s := range siSuffixes {
		if abs >= s.factor {
			return strconv.FormatFloat(v/s.factor, 'f', prec, 64) + s.suffix
		}
	}
	return strconv.FormatFloat(v, 'f', prec, 64)
}

// labelFormatter bridges a Format to the linechart label-formatter hook.
// A zero Format returns nil so charts keep their own defaults.
func (f Format) labelFormatter() linechart.LabelFormatter {
	if f.IsZero() {
		return nil
	}
	return func(_ int, v float64) string { return FormatValue(f, v) }
}
