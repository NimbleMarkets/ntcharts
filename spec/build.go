// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/NimbleMarkets/ntcharts/v2/sparkline"

	"charm.land/lipgloss/v2"
)

// Build converts a Spec into the appropriate ntcharts terminal model.
//
// The concrete return type depends on Spec.Type:
//
//   - ChartTypeBar         -> *barchart.Model (categorical combo: *ComboModel)
//   - ChartTypeLine        -> *linechart.Model (categorical: *CategoricalModel)
//   - ChartTypeTimeSeries  -> *timeserieslinechart.Model
//   - ChartTypeScatter     -> *linechart.Model (categorical: *CategoricalModel)
//   - ChartTypeHeatmap     -> *heatmap.Model
//   - ChartTypeSparkline   -> *sparkline.Model
//   - ChartTypeOHLC        -> *timeserieslinechart.Model
//
// Callers should type-assert the result. Unsupported chart types return an
// error rather than panicking, so future additions to ChartType do not break
// existing callers.
//
// A log axis (XAxis.Scale / YAxis.Scale == ScaleLog) sets the model's own
// scale to linechart.ScaleLog; its ranges stay in data units.
//
// Spec.Y2Axis adds a right-hand Y axis to line, scatter and timeseries
// charts; series with YAxis == YAxisRight are plotted against it. It obeys
// the same pin, scale and format rules as YAxis. A categorical bar/line combo
// also supports Y2Axis; see ResolveCategoricalCombo. Other chart types reject it.
//
// Build does not mutate s.
func Build(s Spec) (any, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if IsCategoricalCombo(s) {
		return buildCombo(s)
	}
	for i, ser := range s.Data.Series {
		if ser.Type != "" && ser.Type != string(s.Type) && !(s.Type == ChartTypeTimeSeries && ser.Type == "line") {
			return nil, fmt.Errorf("spec: series[%d] type %q is unsupported in terminal %q charts", i, ser.Type, s.Type)
		}
	}
	if err := checkY2Support(s); err != nil {
		return nil, err
	}
	if err := checkScaleSupport(s); err != nil {
		return nil, err
	}
	if s.XAxis.Type == XAxisCategory && (s.Type == ChartTypeLine || s.Type == ChartTypeScatter) {
		return buildCategorical(s)
	}
	switch s.Type {
	case ChartTypeBar:
		return buildBar(s)
	case ChartTypeTimeSeries:
		return buildTimeSeries(s)
	case ChartTypeLine:
		return buildLine(s)
	case ChartTypeScatter:
		return buildScatter(s)
	case ChartTypeHeatmap:
		return buildHeatmap(s)
	case ChartTypeSparkline:
		return buildSparkline(s)
	case ChartTypeOHLC:
		return buildOHLC(s)
	case ChartTypeStreamline,
		ChartTypeCanvas:
		return nil, fmt.Errorf("spec: Build for chart type %q is not yet implemented", s.Type)
	default:
		return nil, fmt.Errorf("spec: unknown chart type %q", s.Type)
	}
}

// buildBar constructs a *barchart.Model from s.
//
// The bar chart is built along the following mapping:
//
//	spec.Series              -> stacked BarValue segments within each bar
//	spec.XAxis.Labels        -> per-bar Label
//	spec.YAxis.Max           -> WithMaxValue (if set; otherwise auto-max)
//	spec.Series[i].Color     -> BarValue.Style foreground
//	spec.Theme.Palette       -> fallback colour for series without Color
//	spec.Options.Orientation -> WithHorizontalBars() when OrientationHorizontal
//
// Each X-axis label becomes a single bar whose stacked segments are the
// Y value of every series at the matching index. When XAxis.Labels is empty,
// labels are derived from the first series' DataPoint.X values.
//
// ntcharts' barchart.Model only supports stacked segments within a bar — it
// has no grouped/side-by-side rendering mode. So a Spec with more than one
// Series must set Options.Stacked to true; otherwise Build returns an error
// rather than silently stacking series the caller may have expected to be
// grouped.
func buildBar(s Spec) (*barchart.Model, error) {
	if len(s.Data.Series) > 1 && !s.Options.Stacked {
		return nil, fmt.Errorf("spec: terminal bar charts cannot render grouped bars; set options.stacked=true or use a single series")
	}

	labels := s.XAxis.Labels
	if len(labels) == 0 {
		labels = DeriveBarLabels(s.Data)
	}
	if len(labels) == 0 {
		return nil, fmt.Errorf("spec: bar chart requires x_axis labels or per-series X values")
	}

	data := make([]barchart.BarData, len(labels))
	for i, label := range labels {
		values := make([]barchart.BarValue, 0, len(s.Data.Series))
		for j, ser := range s.Data.Series {
			if i >= len(ser.Values) {
				continue
			}
			values = append(values, barchart.BarValue{
				Name:  ser.Name,
				Value: ser.Values[i].Y,
				Style: seriesStyle(ser, j, s.Theme),
			})
		}
		data[i] = barchart.BarData{Label: label, Values: values}
	}

	if s.YAxis.Min != nil && s.YAxis.Max != nil && *s.YAxis.Min > *s.YAxis.Max {
		return nil, fmt.Errorf("spec: y_axis min %v exceeds max %v", *s.YAxis.Min, *s.YAxis.Max)
	}
	opts := []barchart.Option{barchart.WithDataSet(data)}
	// Each pin goes straight to the model; an unset bound stays under the
	// model's own auto-scaling, which already tracks negative values.
	if s.YAxis.Min != nil {
		opts = append(opts, barchart.WithMinValue(*s.YAxis.Min))
	}
	if s.YAxis.Max != nil {
		opts = append(opts, barchart.WithMaxValue(*s.YAxis.Max))
	}
	if s.Options.Orientation == OrientationHorizontal {
		opts = append(opts, barchart.WithHorizontalBars())
	}

	m := barchart.New(s.Width, s.Height, opts...)
	m.Draw()
	return &m, nil
}

// resolveXFloat resolves a point's numeric X: the point's own X, else the
// shared Data.XAxisData at idx, else the index itself. Shared with buildScatter.
func resolveXFloat(s Spec, p DataPoint, idx int) float64 {
	if x, ok := pointFloat(PointX(p, idx, s.Data.XAxisData)); ok {
		return x
	}
	return float64(idx)
}

// yBounds scans every DataPoint.Y across series and returns the observed
// minimum and maximum. ok is false when no series contained any points.
func yBounds(series []Series) (min, max float64, ok bool) {
	min, max = math.Inf(1), math.Inf(-1)
	for _, ser := range series {
		for _, p := range ser.Values {
			min = math.Min(min, p.Y)
			max = math.Max(max, p.Y)
		}
	}
	return min, max, !math.IsInf(min, 1)
}

// resolveYRange implements the spec-level one-sided Y-axis pin rule for an
// axis: a lone Min or Max pins that bound; the missing bound falls back to
// the data-derived extreme of the axis's series (via yBounds), or 0 when
// there is no data to derive it from. It is an error for the resulting min
// to exceed the resulting max — e.g. a Min pin above the data's actual max
// would silently build an inverted range without this guard.
func resolveYRange(a yAxisRef) (min, max float64, err error) {
	min, max, ok := yBounds(a.series)
	if !ok {
		min, max = 0, 0
	}
	return resolvePinnedYRange(a, min, max)
}

// resolvePinnedYRange preserves explicit bounds when widening a flat range.
// Two equal pins cannot describe a drawable range and are rejected.
func resolvePinnedYRange(a yAxisRef, min, max float64) (float64, float64, error) {
	axis := a.axis
	if axis.Min != nil {
		min = *axis.Min
	}
	if axis.Max != nil {
		max = *axis.Max
	}
	if min > max {
		return 0, 0, fmt.Errorf("spec: %s min %v exceeds max %v", a.name, min, max)
	}
	if min == max {
		if axis.Min != nil && axis.Max != nil {
			return 0, 0, fmt.Errorf("spec: %s min and max must differ", a.name)
		}
		if axis.Min == nil {
			min--
		}
		if axis.Max == nil {
			max++
		}
	}
	return min, max, nil
}

// checkGraphSize runs after ranges and formatters have determined the margins.
func checkGraphSize(m linechart.Model) error {
	if m.GraphWidth() < 1 || m.GraphHeight() < 1 {
		return fmt.Errorf("spec: %d x %d chart is too small for axes and labels", m.Width(), m.Height())
	}
	return nil
}

// yAxisRef is one Y axis of a spec with the series plotted against it.
// name is "y_axis" or "y2_axis" and prefixes every error about the axis,
// so the left and right axes report the same mistakes in the same words.
type yAxisRef struct {
	name   string
	axis   YAxis
	series []Series
}

// splitYAxes returns the series on the left Y axis and on the right one.
func splitYAxes(s Spec) (left, right []Series) {
	for _, ser := range s.Data.Series {
		if ser.YAxis == YAxisRight {
			right = append(right, ser)
		} else {
			left = append(left, ser)
		}
	}
	return left, right
}

// leftAxis returns the spec's left Y axis with its series.
func leftAxis(s Spec) yAxisRef {
	left, _ := splitYAxes(s)
	return yAxisRef{name: "y_axis", axis: s.YAxis, series: left}
}

// rightAxis returns the spec's right Y axis with its series.
// Requires s.Y2Axis != nil.
func rightAxis(s Spec) yAxisRef {
	_, right := splitYAxes(s)
	return yAxisRef{name: "y2_axis", axis: *s.Y2Axis, series: right}
}

// checkY2Support rejects a right Y axis on chart types that cannot draw
// one. A right-hand series without Y2Axis is already a Validate error.
func checkY2Support(s Spec) error {
	if s.Y2Axis == nil {
		return nil
	}
	switch s.Type {
	case ChartTypeLine, ChartTypeScatter, ChartTypeTimeSeries:
		return nil
	}
	return fmt.Errorf("spec: y2_axis is not supported for chart type %q", s.Type)
}

// newY2Axis returns the right Y axis of a chart from its resolved range:
// the spec axis's scale and label format, and the label style of the
// first right-hand series, so the labels read as that series'.
func newY2Axis(s Spec, a yAxisRef, min, max float64) linechart.Axis {
	axis := linechart.Axis{
		Min:            min,
		Max:            max,
		LabelFormatter: axisLabelFormatter(a.axis.Format, a.axis.Scale),
	}
	if a.axis.Scale == ScaleLog {
		axis.Scale = linechart.ScaleLog
	}
	for i, ser := range s.Data.Series {
		if ser.YAxis == YAxisRight {
			axis.LabelStyle = seriesStyle(ser, i, s.Theme)
			break
		}
	}
	return axis
}

// buildLine joins each series' points in input order on a base linechart.
// Numeric X comes from the point, shared X data, or the point's index.
// Linear ranges retain the native defaults (0..1), expanded around the data;
// Y pins follow resolveYRange. Log ranges widen to whole decades.
// Ranges and formatters are resolved before drawing so every segment shares
// the same mapping and series colours, including singleton points.
// A right-hand series (`YAxis == YAxisRight`) is resolved against `Y2Axis`
// with the same rules and drawn through `Y2ToY`.
func buildLine(s Spec) (*linechart.Model, error) {
	logX, logY := s.XAxis.Scale == ScaleLog, s.YAxis.Scale == ScaleLog
	minX, maxX, minY, maxY := 0.0, 1.0, 0.0, 1.0
	var err error
	if logY {
		lo, hi, ok, e := logYBounds(leftAxis(s))
		if e != nil {
			return nil, e
		}
		minY, maxY, err = resolveLogYRange(leftAxis(s), lo, hi, ok)
	} else if s.YAxis.Min != nil || s.YAxis.Max != nil {
		minY, maxY, err = resolveYRange(leftAxis(s))
	} else if lo, hi, ok := yBounds(leftAxis(s).series); ok {
		minY, maxY = math.Min(minY, lo), math.Max(maxY, hi)
	}
	if err != nil {
		return nil, err
	}
	var y2 *linechart.Axis
	if s.Y2Axis != nil {
		a := rightAxis(s)
		minY2, maxY2 := 0.0, 1.0
		if a.axis.Scale == ScaleLog {
			lo, hi, ok, e := logYBounds(a)
			if e != nil {
				return nil, e
			}
			minY2, maxY2, err = resolveLogYRange(a, lo, hi, ok)
		} else if a.axis.Min != nil || a.axis.Max != nil {
			minY2, maxY2, err = resolveYRange(a)
		} else if lo, hi, ok := yBounds(a.series); ok {
			minY2, maxY2 = math.Min(minY2, lo), math.Max(maxY2, hi)
		}
		if err != nil {
			return nil, err
		}
		axis := newY2Axis(s, a, minY2, maxY2)
		y2 = &axis
	}
	if logX {
		lo, hi, ok, e := logXBounds(s)
		if e != nil {
			return nil, e
		}
		if !ok {
			lo, hi = 1, 10
		}
		minX, maxX = logRange(lo, hi, false, false)
	} else {
		for _, ser := range s.Data.Series {
			for j, p := range ser.Values {
				x := resolveXFloat(s, p, j)
				minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			}
		}
	}
	var opts []linechart.Option
	if logX {
		opts = append(opts, linechart.WithXScale(linechart.ScaleLog))
	}
	if logY {
		opts = append(opts, linechart.WithYScale(linechart.ScaleLog))
	}
	if xf := axisLabelFormatter(s.XAxis.Format, s.XAxis.Scale); xf != nil {
		opts = append(opts, linechart.WithXLabelFormatter(xf))
	}
	if yf := axisLabelFormatter(s.YAxis.Format, s.YAxis.Scale); yf != nil {
		opts = append(opts, linechart.WithYLabelFormatter(yf))
	}
	if y2 != nil {
		opts = append(opts, linechart.WithY2Axis(*y2))
	}
	m := linechart.New(s.Width, s.Height, minX, maxX, minY, maxY, opts...)
	if err := checkGraphSize(m); err != nil {
		return nil, err
	}
	m.DrawXYAxisAndLabel()
	for i, ser := range s.Data.Series {
		style := seriesStyle(ser, i, s.Theme)
		y := func(v float64) float64 { return v }
		if ser.YAxis == YAxisRight {
			y = m.Y2ToY
		}
		for j, p := range ser.Values {
			from := canvas.Float64Point{X: resolveXFloat(s, p, j), Y: y(p.Y)}
			to := from // a singleton or final point still gets a braille dot
			if j+1 < len(ser.Values) {
				next := ser.Values[j+1]
				to = canvas.Float64Point{X: resolveXFloat(s, next, j+1), Y: y(next.Y)}
			}
			m.DrawBrailleLineWithStyle(from, to, style)
		}
	}
	return &m, nil
}

// buildScatter renders points onto a base linechart canvas. ntcharts has no
// scatter model; the spec surface owns range computation and point drawing.
// DataPoint.Size is accepted in the schema but ignored by this surface.
// YAxis.Min / YAxis.Max follow the one-sided pin rule (a lone bound pins;
// the other is data-derived); it is an error for the resulting min to
// exceed the resulting max. A right-hand series (`YAxis == YAxisRight`) is
// resolved against `Y2Axis` with the same rules and drawn through `Y2ToY`.
func buildScatter(s Spec) (any, error) {
	logX, logY := s.XAxis.Scale == ScaleLog, s.YAxis.Scale == ScaleLog
	logY2 := s.Y2Axis != nil && s.Y2Axis.Scale == ScaleLog
	type styledPoint struct {
		x, y  float64
		right bool
		style lipgloss.Style
	}
	var pts []styledPoint
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	minY2, maxY2 := math.Inf(1), math.Inf(-1)
	for i, ser := range s.Data.Series {
		st := seriesStyle(ser, i, s.Theme)
		right := ser.YAxis == YAxisRight
		axisName, logAxis := "y_axis", logY
		if right {
			axisName, logAxis = "y2_axis", logY2
		}
		for j, p := range ser.Values {
			x := resolveXFloat(s, p, j)
			if logX && !logOK(x) {
				return nil, logValueError("x_axis", fmt.Sprintf("series %q point %d has x", ser.Name, j), x)
			}
			if logAxis && !logOK(p.Y) {
				return nil, logValueError(axisName, fmt.Sprintf("series %q point %d has y", ser.Name, j), p.Y)
			}
			pts = append(pts, styledPoint{x: x, y: p.Y, right: right, style: st})
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			if right {
				minY2, maxY2 = math.Min(minY2, p.Y), math.Max(maxY2, p.Y)
			} else {
				minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
			}
		}
	}
	if len(pts) == 0 {
		return nil, fmt.Errorf("spec: scatter requires at least one data point")
	}
	okY := !math.IsInf(minY, 1)
	if !okY { // no left-hand points
		minY, maxY = 0, 0
	}
	var err error
	if logY {
		minY, maxY, err = resolveLogYRange(leftAxis(s), minY, maxY, okY)
	} else {
		minY, maxY, err = resolvePinnedYRange(leftAxis(s), minY, maxY)
	}
	if err != nil {
		return nil, err
	}
	var y2 *linechart.Axis
	if s.Y2Axis != nil {
		a := rightAxis(s)
		ok := !math.IsInf(minY2, 1)
		if !ok {
			minY2, maxY2 = 0, 0
		}
		var e error
		if logY2 {
			minY2, maxY2, e = resolveLogYRange(a, minY2, maxY2, ok)
		} else {
			minY2, maxY2, e = resolvePinnedYRange(a, minY2, maxY2)
		}
		if e != nil {
			return nil, e
		}
		axis := newY2Axis(s, a, minY2, maxY2)
		y2 = &axis
	}
	if logX {
		minX, maxX = logRange(minX, maxX, false, false)
	} else if minX == maxX {
		minX, maxX = minX-1, maxX+1
	}
	var opts []linechart.Option
	if logX {
		opts = append(opts, linechart.WithXScale(linechart.ScaleLog))
	}
	if logY {
		opts = append(opts, linechart.WithYScale(linechart.ScaleLog))
	}
	if xf := axisLabelFormatter(s.XAxis.Format, s.XAxis.Scale); xf != nil {
		opts = append(opts, linechart.WithXLabelFormatter(xf))
	}
	if yf := axisLabelFormatter(s.YAxis.Format, s.YAxis.Scale); yf != nil {
		opts = append(opts, linechart.WithYLabelFormatter(yf))
	}
	if y2 != nil {
		opts = append(opts, linechart.WithY2Axis(*y2))
	}
	m := linechart.New(s.Width, s.Height, minX, maxX, minY, maxY, opts...)
	if err := checkGraphSize(m); err != nil {
		return nil, err
	}
	m.DrawXYAxisAndLabel()
	for _, p := range pts {
		y := p.y
		if p.right {
			y = m.Y2ToY(y)
		}
		m.DrawRuneWithStyle(canvas.Float64Point{X: p.x, Y: y}, '•', p.style)
	}
	return &m, nil
}

// buildTimeSeries constructs a *timeserieslinechart.Model from s.
//
// Implementation notes and TODOs for contributors extending this:
//
//  1. Each spec.Series becomes a named data set in the timeserieslinechart
//     (see SetDataSetStyle / PushDataSet).
//  2. DataPoint.X accepts time.Time or RFC3339 strings or numeric ms; the
//     helper PointTime() converts all three.
//  3. YAxis.Min / YAxis.Max pin the Y axis via WithYRange following the
//     one-sided pin rule: a lone Min or Max pins that bound and the missing
//     bound is derived from the pushed points; when both are nil the chart
//     auto-scales.
//  4. YAxis.Format / XAxis.Format (when Kind == "time") thread custom
//     LabelFormatters through WithYLabelFormatter / WithXLabelFormatter at
//     construction time; when unset, the chart's own defaults (including
//     DateTimeLabelFormatter for X) are used.
//  5. Per-series Color is applied via SetDataSetStyle using a lipgloss
//     foreground. Background and gridlines are not rendered.
//  6. Y2Axis adds a right Y axis with the same range rules; series with
//     YAxis == YAxisRight are assigned to it with SetDataSetYAxis before
//     their points are pushed, so their data auto-ranges the right axis.
//
// This implementation covers the common case: one or more time-indexed
// series with a shared auto-ranged axis. Streamlined and stacked variants
// are left for future extensions.
func buildTimeSeries(s Spec) (*timeserieslinechart.Model, error) {
	tMin, tMax, ok := timeBounds(s.Data)
	if !ok {
		return nil, fmt.Errorf("spec: timeseries chart requires at least one DataPoint with a time X value")
	}
	tMin, tMax = drawableTimeRange(tMin, tMax)

	opts := []timeserieslinechart.Option{
		timeserieslinechart.WithTimeRange(tMin, tMax),
	}
	logY := s.YAxis.Scale == ScaleLog
	if logY {
		minY, maxY, ok, err := logYBounds(leftAxis(s))
		if err != nil {
			return nil, err
		}
		if minY, maxY, err = resolveLogYRange(leftAxis(s), minY, maxY, ok); err != nil {
			return nil, err
		}
		opts = append(opts,
			timeserieslinechart.WithYScale(linechart.ScaleLog),
			timeserieslinechart.WithYRange(minY, maxY))
	} else if s.YAxis.Min != nil || s.YAxis.Max != nil {
		minY, maxY, err := resolveYRange(leftAxis(s))
		if err != nil {
			return nil, err
		}
		opts = append(opts, timeserieslinechart.WithYRange(minY, maxY))
	}
	if yf := axisLabelFormatter(s.YAxis.Format, s.YAxis.Scale); yf != nil {
		opts = append(opts, timeserieslinechart.WithYLabelFormatter(yf))
	}
	if s.Y2Axis != nil {
		a := rightAxis(s)
		logY2 := a.axis.Scale == ScaleLog
		minY2, maxY2 := 0.0, 1.0 // the model's own default, widened by auto-ranging
		if logY2 {
			lo, hi, ok, err := logYBounds(a)
			if err != nil {
				return nil, err
			}
			if minY2, maxY2, err = resolveLogYRange(a, lo, hi, ok); err != nil {
				return nil, err
			}
		} else if a.axis.Min != nil || a.axis.Max != nil {
			var err error
			if minY2, maxY2, err = resolveYRange(a); err != nil {
				return nil, err
			}
		}
		axis := newY2Axis(s, a, minY2, maxY2)
		axis.AutoMin = !logY2 && a.axis.Min == nil
		axis.AutoMax = !logY2 && a.axis.Max == nil
		opts = append(opts, timeserieslinechart.WithY2Axis(axis))
	}
	// Only kind == "time" X formats apply to a timeseries X axis; other
	// kinds (number/percent/currency/si) are intentionally ignored here
	// since the X axis values are time, not the quantity those kinds format.
	if !s.XAxis.Format.IsZero() && s.XAxis.Format.Kind == "time" {
		// timeserieslinechart's default DateTimeLabelFormatter (see
		// linechart/timeserieslinechart/timeserieslinechart.go) receives X
		// label values as seconds-since-epoch (it does
		// time.UnixMilli(int64(math.Round(v*1e3)))); mirror that conversion
		// exactly here, only swapping in the custom layout.
		layout := s.XAxis.Format.Layout
		if layout == "" {
			layout = "2006-01-02"
		}
		opts = append(opts, timeserieslinechart.WithXLabelFormatter(
			makeTimeAxisFormatter(layout)))
	}

	m := timeserieslinechart.New(s.Width, s.Height, opts...)
	m.AutoMinY = !logY && s.YAxis.Min == nil
	m.AutoMaxY = !logY && s.YAxis.Max == nil

	for i, ser := range s.Data.Series {
		name := ser.Name
		if ser.YAxis == YAxisRight {
			m.SetDataSetYAxis(name, linechart.YAxisRight)
		}
		m.SetDataSetStyle(name, seriesStyle(ser, i, s.Theme))

		for j, p := range ser.Values {
			t, ok := PointTime(PointX(p, j, s.Data.XAxisData))
			if !ok {
				continue
			}
			m.PushDataSet(name, timeserieslinechart.TimePoint{Time: t, Value: p.Y})
		}
	}

	if err := checkGraphSize(m.Model); err != nil {
		return nil, err
	}
	m.DrawBrailleAll()
	return &m, nil
}

// makeTimeAxisFormatter returns a linechart.LabelFormatter for the X axis of
// a timeserieslinechart that renders the given Go time layout instead of the
// package default DateTimeLabelFormatter's "MM/DD" / "'YY MM/DD" scheme.
//
// timeserieslinechart passes X label values as seconds-since-epoch (see
// DateTimeLabelFormatter in linechart/timeserieslinechart/timeserieslinechart.go,
// which computes time.UnixMilli(int64(math.Round(v * 1e3)))); this mirrors
// that value->time conversion exactly, only swapping in a custom layout.
func makeTimeAxisFormatter(layout string) linechart.LabelFormatter {
	return func(_ int, v float64) string {
		t := time.UnixMilli(int64(math.Round(v * 1e3))).UTC()
		return t.Format(layout)
	}
}

// DeriveBarLabels returns the category labels for a bar chart whose Spec did
// not set XAxis.Labels: the first series' DataPoint.X values, falling back
// to Data.XAxisData for omitted X values, then the 1-based point index
// where the resolved X is not a string.
// Every surface that renders bar charts (Build here, ToECharts in the
// spec/echarts module) applies this same rule so labels agree across them.
func DeriveBarLabels(data Data) []string {
	if len(data.Series) == 0 {
		return nil
	}
	first := data.Series[0].Values
	labels := make([]string, 0, len(first))
	for i, p := range first {
		if s, ok := pointString(PointX(p, i, data.XAxisData)); ok {
			labels = append(labels, s)
			continue
		}
		labels = append(labels, fmt.Sprintf("%d", i+1))
	}
	return labels
}

// timeBounds scans every resolved DataPoint.X and returns the minimum and
// maximum time.Time. The third return value is false if no series contained
// a time-valued X.
func timeBounds(data Data) (time.Time, time.Time, bool) {
	var (
		min, max time.Time
		found    bool
	)
	for _, ser := range data.Series {
		for i, p := range ser.Values {
			t, ok := PointTime(PointX(p, i, data.XAxisData))
			if !ok {
				continue
			}
			if !found {
				min, max = t, t
				found = true
				continue
			}
			if t.Before(min) {
				min = t
			}
			if t.After(max) {
				max = t
			}
		}
	}
	return min, max, found
}

// The timeseries model stores millisecond timestamps. Widen singleton
// ranges, including distinct sub-millisecond times that collapse to one value.
func drawableTimeRange(min, max time.Time) (time.Time, time.Time) {
	if max.UnixMilli() <= min.UnixMilli() {
		center := time.UnixMilli(min.UnixMilli())
		return center.Add(-12 * time.Hour), center.Add(12 * time.Hour)
	}
	return min, max
}

// buildHeatmap renders Heat data onto a heatmap model. Theme.Gradient (hex
// stops) becomes an interpolated color scale; absent, the package default
// grayscale applies.
func buildHeatmap(s Spec) (any, error) {
	if s.Heat == nil || (len(s.Heat.Cells) == 0 && len(s.Heat.Matrix) == 0) {
		return nil, fmt.Errorf("spec: heatmap requires Heat data (cells or matrix)")
	}
	var opts []heatmap.Option
	cs, err := gradientScale(s.Theme.Gradient)
	if err != nil {
		return nil, err
	}
	if cs != nil {
		opts = append(opts, heatmap.WithColorScale(cs))
	}
	if s.Heat.MinValue != nil || s.Heat.MaxValue != nil {
		minValue, maxValue := math.Inf(1), math.Inf(-1)
		observe := func(v float64) {
			minValue = math.Min(minValue, v)
			maxValue = math.Max(maxValue, v)
		}
		if len(s.Heat.Matrix) > 0 {
			for _, row := range s.Heat.Matrix {
				for _, v := range row {
					observe(v)
				}
			}
		} else {
			for _, cell := range s.Heat.Cells {
				observe(cell.Z)
			}
		}
		if math.IsInf(minValue, 1) {
			return nil, fmt.Errorf("spec: heatmap requires at least one cell")
		}
		if s.Heat.MinValue != nil {
			minValue = *s.Heat.MinValue
		}
		if s.Heat.MaxValue != nil {
			maxValue = *s.Heat.MaxValue
		}
		if minValue > maxValue {
			return nil, fmt.Errorf("spec: heat min_value %v exceeds max_value %v", minValue, maxValue)
		}
		opts = append(opts, heatmap.WithValueRange(minValue, maxValue))
	} else {
		opts = append(opts, heatmap.WithAutoValueRange())
	}
	// The cells form an index grid, drawn as filled blocks that tile the plot.
	// Cell (x, y) is column x and row y, with row 0 first: the topmost row,
	// where y_axis.labels[0] is drawn, as in flint's own heatmaps. The model
	// counts rows up from the bottom, so rows and row labels are mirrored.
	var cells []HeatCell
	if len(s.Heat.Matrix) > 0 {
		for y, row := range s.Heat.Matrix {
			for x, value := range row {
				cells = append(cells, HeatCell{X: float64(x), Y: float64(y), Z: value})
			}
		}
	} else {
		cells = s.Heat.Cells
	}
	nx, ny := len(s.XAxis.Labels), len(s.YAxis.Labels)
	for _, c := range cells {
		if c.X < 0 || c.Y < 0 {
			return nil, fmt.Errorf("spec: heatmap cell indices must be non-negative; got cell (%v, %v)", c.X, c.Y)
		}
		nx, ny = max(nx, int(math.Floor(c.X))+1), max(ny, int(math.Floor(c.Y))+1)
	}
	opts = append(opts, heatmap.WithCellSize(1, 1))
	m := heatmap.New(s.Width, s.Height, opts...)
	if len(s.XAxis.Labels) == 0 {
		m.SetXStep(0) // no column labels: no rows reserved for them
	}
	if len(s.YAxis.Labels) == 0 {
		m.SetYStep(0) // no row labels: no margin reserved for them
	}
	xLabels := make([]string, nx)
	copy(xLabels, s.XAxis.Labels)
	yLabels := make([]string, ny)
	for j := range yLabels {
		if i := ny - 1 - j; i < len(s.YAxis.Labels) {
			yLabels[j] = s.YAxis.Labels[i]
		}
	}
	m.SetXLabels(xLabels)
	m.SetYLabels(yLabels)
	for _, c := range cells {
		m.Push(heatmap.NewHeatPoint(c.X, float64(ny-1)-c.Y, c.Z))
	}
	m.Draw()
	return &m, nil
}

// buildSparkline renders a single series onto a sparkline model. The
// sparkline package is single-series and has no Y-minimum: multiple series
// error (like grouped bars), YAxis.Min is documented-ignored, YAxis.Max maps
// to WithMaxValue.
func buildSparkline(s Spec) (any, error) {
	if len(s.Data.Series) != 1 {
		return nil, fmt.Errorf("spec: sparkline supports exactly one series; got %d", len(s.Data.Series))
	}
	ser := s.Data.Series[0]
	opts := []sparkline.Option{sparkline.WithStyle(seriesStyle(ser, 0, s.Theme))}
	if s.YAxis.Max != nil {
		opts = append(opts, sparkline.WithMaxValue(*s.YAxis.Max))
	}
	m := sparkline.New(s.Width, s.Height, opts...)
	for _, p := range ser.Values {
		m.Push(p.Y)
	}
	m.Draw()
	return &m, nil
}

// Default up/down candle colors (overridden by Theme.Palette slots 0/1).
const (
	defaultUpColor   = "#26a69a"
	defaultDownColor = "#ef5350"
)

// buildOHLC constructs a *timeserieslinechart.Model that renders
// Series.OHLC candlesticks at time-scaled X positions across the full
// graph width, with labeled axes (the same axis machinery as
// ChartTypeTimeSeries). The Y range derives from the data's low/high and
// may be pinned by YAxis.Min/Max (one-sided pins allowed; it is an error
// for the resulting min to exceed the max). Candle body width is chosen
// automatically from chart density (see autoCandleWidth); dense data
// overdraws like any dense timeseries — no points are dropped.
func buildOHLC(s Spec) (any, error) {
	if len(s.Data.Series) != 1 {
		return nil, fmt.Errorf("spec: ohlc supports exactly one series; got %d", len(s.Data.Series))
	}
	pts := s.Data.Series[0].OHLC
	if len(pts) == 0 {
		return nil, fmt.Errorf("spec: ohlc requires Series.OHLC points")
	}

	block := false
	switch s.Options.CandleStyle {
	case "", CandleStyleLine:
	case CandleStyleBlock:
		block = true
	default:
		return nil, fmt.Errorf("spec: unknown candle_style %q", s.Options.CandleStyle)
	}

	times := make([]time.Time, len(pts))
	for i, p := range pts {
		t, ok := PointTime(p.T)
		if !ok {
			return nil, fmt.Errorf("spec: ohlc point %d has unparsable time %v", i, p.T)
		}
		times[i] = t
	}
	tMin, tMax := times[0], times[0]
	for _, t := range times[1:] {
		if t.Before(tMin) {
			tMin = t
		}
		if t.After(tMax) {
			tMax = t
		}
	}
	tMin, tMax = drawableTimeRange(tMin, tMax)

	logY := s.YAxis.Scale == ScaleLog
	minP, maxP := math.Inf(1), math.Inf(-1)
	for i, p := range pts {
		if logY {
			for _, f := range []struct {
				name string
				v    float64
			}{{"open", p.O}, {"high", p.H}, {"low", p.L}, {"close", p.C}} {
				if !logOK(f.v) {
					return nil, logValueError("y_axis", fmt.Sprintf("ohlc point %d has %s", i, f.name), f.v)
				}
			}
		}
		minP = math.Min(minP, p.L)
		maxP = math.Max(maxP, p.H)
	}
	var err error
	if logY {
		minP, maxP, err = resolveLogYRange(leftAxis(s), minP, maxP, true)
	} else {
		minP, maxP, err = resolvePinnedYRange(leftAxis(s), minP, maxP)
	}
	if err != nil {
		return nil, err
	}

	opts := []timeserieslinechart.Option{timeserieslinechart.WithTimeRange(tMin, tMax)}
	if logY {
		opts = append(opts, timeserieslinechart.WithYScale(linechart.ScaleLog))
	}
	opts = append(opts, timeserieslinechart.WithYRange(minP, maxP))
	if yf := axisLabelFormatter(s.YAxis.Format, s.YAxis.Scale); yf != nil {
		opts = append(opts, timeserieslinechart.WithYLabelFormatter(yf))
	}
	if !s.XAxis.Format.IsZero() && s.XAxis.Format.Kind == "time" {
		layout := s.XAxis.Format.Layout
		if layout == "" {
			layout = "2006-01-02"
		}
		opts = append(opts, timeserieslinechart.WithXLabelFormatter(
			makeTimeAxisFormatter(layout)))
	}

	m := timeserieslinechart.New(s.Width, s.Height, opts...)
	m.AutoMinY = !logY && s.YAxis.Min == nil
	m.AutoMaxY = !logY && s.YAxis.Max == nil
	for i, p := range pts {
		ts := times[i]
		m.PushDataSet("open", timeserieslinechart.TimePoint{Time: ts, Value: p.O})
		m.PushDataSet("high", timeserieslinechart.TimePoint{Time: ts, Value: p.H})
		m.PushDataSet("low", timeserieslinechart.TimePoint{Time: ts, Value: p.L})
		m.PushDataSet("close", timeserieslinechart.TimePoint{Time: ts, Value: p.C})
	}

	up := candleStyle(s.Theme, 0, defaultUpColor)
	down := candleStyle(s.Theme, 1, defaultDownColor)

	if err := checkGraphSize(m.Model); err != nil {
		return nil, err
	}
	m.DrawCandleWithOpts("open", "high", "low", "close", up, down,
		timeserieslinechart.DrawCandleOpts{
			Width: autoCandleWidth(m.GraphWidth(), times, tMin, tMax),
			Block: block,
		})
	return &m, nil
}

// autoCandleWidth picks a candle body width (in columns, odd, >= 1) from
// chart density: roughly an equal share of the graph width per candle,
// clamped so adjacent time-scaled candles cannot overlap on irregular
// calendars, and capped at 7 so sparse charts stay recognizable candles.
func autoCandleWidth(graphWidth int, times []time.Time, tMin, tMax time.Time) int {
	n := len(times)
	if n == 0 || graphWidth < 1 {
		return 1
	}
	w := graphWidth/n - 1
	if n > 1 && tMax.After(tMin) {
		span := float64(tMax.Sub(tMin))
		cols := make([]int, n)
		for i, t := range times {
			// Match the draw path's column formula exactly (see
			// DrawCandleWithOpts / dataSet.tBuf scaling in
			// linechart/timeserieslinechart/timeserieslinechart.go):
			// int(...) truncates a scaled-by-graphWidth fraction, it does
			// not round a scaled-by-(graphWidth-1) fraction.
			cols[i] = int(float64(t.Sub(tMin)) / span * float64(graphWidth))
		}
		sort.Ints(cols)
		minGap := graphWidth
		for i := 1; i < n; i++ {
			if g := cols[i] - cols[i-1]; g > 0 && g < minGap {
				minGap = g
			}
		}
		if w > minGap-1 {
			w = minGap - 1
		}
	}
	if w > 7 {
		w = 7
	}
	if w%2 == 0 {
		w--
	}
	if w < 1 {
		w = 1
	}
	return w
}

// candleStyle returns the lipgloss.Style for a candle, preferring
// Theme.Palette[slot] and falling back to fallback (a hex colour) when the
// palette does not have that slot set.
func candleStyle(t Theme, slot int, fallback string) lipgloss.Style {
	color := fallback
	if slot < len(t.Palette) && t.Palette[slot] != "" {
		color = t.Palette[slot]
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

// seriesStyle returns the lipgloss.Style for a Series, preferring the
// series' own Color, then the Theme palette (by index), then the default.
func seriesStyle(ser Series, index int, theme Theme) lipgloss.Style {
	colour := ser.Color
	if colour == "" && index < len(theme.Palette) {
		colour = theme.Palette[index]
	}
	if colour == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colour))
}
