// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"math"
)

// IsCategoricalCombo reports whether a bar spec requests a series type override.
// ResolveCategoricalCombo validates that the override is supported.
func IsCategoricalCombo(s Spec) bool {
	if s.Type != ChartTypeBar {
		return false
	}
	for _, ser := range s.Data.Series {
		if ser.Type != "" && ser.Type != "bar" {
			return true
		}
	}
	return false
}

// CategoricalCombo is the resolved, shared input for terminal and web renderers.
// Values are indexed by Labels; nil means missing, not zero. Series indexes
// refer to the original spec, preserving palette order even if line comes first.
type CategoricalCombo struct {
	Labels               []string
	Bars, Line           []*float64
	BarIndex, LineIndex  int
	Min, Max, Min2, Max2 float64
}

// ResolveCategoricalCombo accepts a parent bar chart with exactly one left bar
// and one right line, a Y2Axis, and linear categorical axes. Labels are required
// explicitly (XAxis.Labels or Data.XAxisData); sparse points name their category.
// It does not mutate s. Callers should Validate s first.
func ResolveCategoricalCombo(s Spec) (CategoricalCombo, error) {
	c := CategoricalCombo{BarIndex: -1, LineIndex: -1}
	fail := func(reason string) (CategoricalCombo, error) {
		return c, fmt.Errorf("spec: categorical combo: %s", reason)
	}
	if s.Type != ChartTypeBar || len(s.Data.Series) != 2 || s.Y2Axis == nil {
		return fail("requires a bar spec, one left bar, one right line, and y2_axis")
	}
	if s.XAxis.Type != "" && s.XAxis.Type != XAxisCategory {
		return fail("x_axis must be category")
	}
	if s.Options.Stacked || s.Options.Orientation == OrientationHorizontal {
		return fail("stacking and horizontal bars are unsupported")
	}
	for _, scale := range []string{s.XAxis.Scale, s.YAxis.Scale, s.Y2Axis.Scale} {
		if scale != "" && scale != ScaleLinear {
			return fail("only linear scales are supported")
		}
	}
	for i, ser := range s.Data.Series {
		typ := ser.Type
		if typ == "" {
			typ = string(s.Type)
		}
		switch {
		case typ == "bar" && ser.YAxis != YAxisRight && c.BarIndex < 0:
			c.BarIndex = i
		case typ == "line" && ser.YAxis == YAxisRight && c.LineIndex < 0:
			c.LineIndex = i
		default:
			return fail("requires exactly one left bar and one right line")
		}
	}
	if c.BarIndex < 0 || c.LineIndex < 0 {
		return fail("requires exactly one left bar and one right line")
	}
	c.Labels = append([]string(nil), s.XAxis.Labels...)
	if len(c.Labels) == 0 {
		for _, x := range s.Data.XAxisData {
			label, ok := x.(string)
			if !ok {
				return fail("shared categories must be strings")
			}
			c.Labels = append(c.Labels, label)
		}
	}
	if len(c.Labels) == 0 {
		return fail("requires ordered x_axis.labels or data.x_axis_data")
	}
	indexes := make(map[string]int, len(c.Labels))
	for i, label := range c.Labels {
		if _, ok := indexes[label]; ok {
			return fail("duplicate category " + label)
		}
		indexes[label] = i
	}
	values := make([][]*float64, 2)
	for i, ser := range s.Data.Series {
		values[i] = make([]*float64, len(c.Labels))
		for j, p := range ser.Values {
			if math.IsNaN(p.Y) || math.IsInf(p.Y, 0) {
				return fail("Y values must be finite; omit a point to represent missing data")
			}
			idx := j
			if x := PointX(p, j, s.Data.XAxisData); x != nil {
				label, ok := x.(string)
				if !ok {
					return fail("point categories must be strings")
				}
				idx, ok = indexes[label]
				if !ok {
					return fail("unknown category " + label)
				}
			}
			if idx >= len(c.Labels) {
				return fail("more positional values than categories")
			}
			if values[i][idx] != nil {
				return fail("duplicate point for category " + c.Labels[idx])
			}
			y := p.Y
			values[i][idx] = &y
		}
	}
	c.Bars, c.Line = values[c.BarIndex], values[c.LineIndex]
	for _, a := range []yAxisRef{leftAxis(s), rightAxis(s)} {
		for _, pin := range []*float64{a.axis.Min, a.axis.Max} {
			if pin != nil && (math.IsNaN(*pin) || math.IsInf(*pin, 0)) {
				return fail("axis bounds must be finite")
			}
		}
	}
	lo, hi, ok := yBounds(leftAxis(s).series)
	if !ok {
		lo, hi = 0, 0
	}
	var err error
	c.Min, c.Max, err = resolvePinnedYRange(leftAxis(s), math.Min(0, lo), math.Max(0, hi))
	if err != nil {
		return c, err
	}
	if c.Min > 0 || c.Max < 0 {
		return fail("bar range must include zero")
	}
	c.Min2, c.Max2, err = resolveYRange(rightAxis(s))
	if err != nil {
		return c, err
	}
	if math.IsInf(c.Max-c.Min, 0) || math.IsInf(c.Max2-c.Min2, 0) {
		return fail("axis span must be finite")
	}
	return c, nil
}

// ComboSeriesColor returns a series colour, using the palette in input order
// and distinct defaults when no colour is supplied.
func ComboSeriesColor(s Spec, index int) string {
	if v := s.Data.Series[index].Color; v != "" {
		return v
	}
	if index < len(s.Theme.Palette) && s.Theme.Palette[index] != "" {
		return s.Theme.Palette[index]
	}
	if s.Data.Series[index].Type == "line" {
		return "#f6ad55"
	}
	return "#63b3ed"
}
