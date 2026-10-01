// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"math"
	"strconv"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// A log axis is drawn by the chart models themselves (linechart.ScaleLog).
// This file holds what the spec adds on top: which chart types may have
// one, the hard errors for values a log axis cannot place, the whole-decade
// range, and the default label format.

// checkScaleSupport rejects a log axis on chart types that cannot draw one.
// Drawing those charts linearly would misrepresent the data.
func checkScaleSupport(s Spec) error {
	logX, logY := s.XAxis.Scale == ScaleLog, s.YAxis.Scale == ScaleLog
	if !logX && !logY {
		return nil
	}
	axis := "y_axis"
	if logX {
		axis = "x_axis"
	}
	var why string
	switch s.Type {
	case ChartTypeBar:
		why = "bars are measured linearly from a zero baseline"
	case ChartTypeHeatmap:
		why = "its axes are cell indices, not scaled values"
	case ChartTypeSparkline:
		why = "columns are measured linearly from zero and it draws no axis"
	case ChartTypeTimeSeries, ChartTypeOHLC:
		if !logX {
			return nil
		}
		why = "its X axis is time"
	default:
		return nil
	}
	return fmt.Errorf("spec: %s chart cannot draw %s scale %q: %s", s.Type, axis, ScaleLog, why)
}

// logOK reports whether v can sit on a log axis.
func logOK(v float64) bool {
	return v > 0 && !math.IsInf(v, 1)
}

// logValueError names a value a log axis cannot place. where describes it,
// e.g. `series "a" point 3 has y`.
func logValueError(axis, where string, v float64) error {
	return fmt.Errorf("spec: %s scale %q requires positive finite values; %s = %v", axis, ScaleLog, where, v)
}

// logYBounds is yBounds for a log Y axis: it returns the observed minimum
// and maximum Y, or an error naming the first value the axis cannot place.
func logYBounds(series []Series) (min, max float64, ok bool, err error) {
	for _, ser := range series {
		for i, p := range ser.Values {
			if !logOK(p.Y) {
				return 0, 0, false, logValueError("y_axis", fmt.Sprintf("series %q point %d has y", ser.Name, i), p.Y)
			}
		}
	}
	min, max, ok = yBounds(series)
	return min, max, ok, nil
}

// logXBounds is the X counterpart of logYBounds, over each point's resolved
// numeric X (see resolveXFloat).
func logXBounds(s Spec) (min, max float64, ok bool, err error) {
	min, max = math.Inf(1), math.Inf(-1)
	for _, ser := range s.Data.Series {
		for i, p := range ser.Values {
			x := resolveXFloat(s, p, i)
			if !logOK(x) {
				return 0, 0, false, logValueError("x_axis", fmt.Sprintf("series %q point %d has x", ser.Name, i), x)
			}
			min, max = math.Min(min, x), math.Max(max, x)
		}
	}
	return min, max, !math.IsInf(min, 1), nil
}

// logRange widens a range of positive data values for a log axis. A pinned
// bound is used exactly; an unpinned one moves out to a power of ten, so
// the axis ends on a whole decade. If that leaves no range (every value on
// one power of ten), each unpinned bound moves out a further decade.
func logRange(min, max float64, pinMin, pinMax bool) (float64, float64) {
	// math.Log10(1000) is 2.9999999999999996: snap before taking the floor
	decade := func(v float64) float64 {
		p := math.Log10(v)
		if r := math.Round(p); math.Abs(p-r) < 1e-9 {
			return r
		}
		return p
	}
	lo, hi := decade(min), decade(max)
	if !pinMin {
		lo = math.Floor(lo)
	}
	if !pinMax {
		hi = math.Ceil(hi)
	}
	if lo >= hi {
		if !pinMin {
			lo--
		}
		if !pinMax {
			hi++
		}
	}
	if !pinMin {
		min = math.Pow10(int(lo))
	}
	if !pinMax {
		max = math.Pow10(int(hi))
	}
	return min, max
}

// resolveLogYRange is resolvePinnedYRange for a log axis. min and max are
// the data's extremes and the pins are data values, so the one-sided pin
// rule and its errors are evaluated on real values, and so is the returned
// range (see logRange). With no data (ok false) an unpinned bound sits one decade
// from the pinned one, or the range is 1..10.
func resolveLogYRange(axis YAxis, min, max float64, ok bool) (float64, float64, error) {
	if axis.Min != nil && !logOK(*axis.Min) {
		return 0, 0, fmt.Errorf("spec: y_axis scale %q requires a positive finite min; got %v", ScaleLog, *axis.Min)
	}
	if axis.Max != nil && !logOK(*axis.Max) {
		return 0, 0, fmt.Errorf("spec: y_axis scale %q requires a positive finite max; got %v", ScaleLog, *axis.Max)
	}
	if !ok {
		min, max = 1, 10
		switch {
		case axis.Min != nil:
			max = *axis.Min * 10
		case axis.Max != nil:
			min = *axis.Max / 10
		}
	}
	if axis.Min != nil {
		min = *axis.Min
	}
	if axis.Max != nil {
		max = *axis.Max
	}
	if min > max {
		return 0, 0, fmt.Errorf("spec: y_axis min %v exceeds max %v", min, max)
	}
	if min == max && axis.Min != nil && axis.Max != nil {
		return 0, 0, fmt.Errorf("spec: y_axis min and max must differ")
	}
	min, max = logRange(min, max, axis.Min != nil, axis.Max != nil)
	return min, max, nil
}

// axisLabelFormatter returns the label formatter for an axis:
// f.labelFormatter, which is nil for a zero Format so charts keep their own
// default — except on a log axis, where the default is formatLogValue,
// since the charts' own (a whole number) would print every tick below 1 as 0.
func axisLabelFormatter(f Format, scale string) linechart.LabelFormatter {
	if scale == ScaleLog && f.IsZero() {
		return func(_ int, v float64) string { return formatLogValue(v) }
	}
	return f.labelFormatter()
}

// formatLogValue is the default label for a log axis, kept short across many
// decades: three significant digits, trailing zeros dropped, with a k/M/G/T
// suffix from 1000 up ("1", "31.6", "0.001", "1k", "3.16M"). Values under
// 0.0001 or from 1e15 up fall back to exponent notation ("1e-05").
func formatLogValue(v float64) string {
	v, _ = strconv.ParseFloat(strconv.FormatFloat(v, 'e', 2, 64), 64)
	abs := math.Abs(v)
	if abs != 0 && (abs < 1e-4 || abs >= 1e15) {
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
	for _, s := range siSuffixes {
		if abs >= s.factor {
			return strconv.FormatFloat(v/s.factor, 'f', -1, 64) + s.suffix
		}
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
