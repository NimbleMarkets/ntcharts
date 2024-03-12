// Package timeserieslinechart implements a linechart that draws lines
// for time series data points
package timeserieslinechart

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/NimbleMarkets/bubbletea-charts/canvas"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/buffer"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/graph"
	"github.com/NimbleMarkets/bubbletea-charts/canvas/runes"
	"github.com/NimbleMarkets/bubbletea-charts/linechart"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const DefaultDataSetName = "default"

func DateTimeLabelFormatter() linechart.LabelFormatter {
	var yearLabel string
	return func(i int, v float64) string {
		if i == 0 { // reset year labeling if redisplaying values
			yearLabel = ""
		}
		t := time.Unix(int64(v), 0).UTC()
		monthDay := t.Format("01/02")
		year := t.Format("'06")
		if yearLabel != year { // apply year label if first time seeing year
			yearLabel = year
			return fmt.Sprintf("%s %s", yearLabel, monthDay)
		} else {
			return monthDay
		}
	}
}

func HourTimeLabelFormatter() linechart.LabelFormatter {
	return func(i int, v float64) string {
		t := time.Unix(int64(v), 0).UTC()
		return t.Format("15:04:05")
	}
}

type TimePoint struct {
	Time  time.Time
	Value float64
}

type dataSet struct {
	LineStyle runes.LineStyle // type of line runes to draw
	Style     lipgloss.Style

	lastTime time.Time // last seen time value

	// stores TimePoints as FloatPoint64{X:time.Time, Y: value} used to draw line runes
	// time.Time will be converted to seconds since epoch for storage
	tBuf *buffer.Float64PointScaleBuffer

	// stores data point values mapped to a specific column of the canvas
	// used to obtain average of point values to display
	// should always been the same size as the Model timeBuckets
	points [][]float64
}

// Model contains state of a timeserieslinechart with an embedded linechart.Model
// The X axis contains time.Time values and the Y axis contains float64 values.
// A data set consists of a sequence TimePoints in chronological order.
// If multiple TimePoints map to the same column, then average value the time points
// will be used as the Y value of the column.
// The X axis contains a time range and the Y axis contains a numeric value range.
// Uses linechart Model UpdateHandler() for processing keyboard and mouse messages.
type Model struct {
	linechart.Model
	dLineStyle runes.LineStyle     // default data set LineStyletype
	dStyle     lipgloss.Style      // default data set Style
	dSets      map[string]*dataSet // maps names to data sets

	// chunk the time ranges into buckets with total size of graphing area
	// each value represents the start time of each time slice for a graph column
	timeBuckets []time.Time
}

// New returns a timeserieslinechart Model initialized from
// width, height, Y value range and various options.
// By default, the chart will set time.Now() as the minimum time,
// enable auto set X and Y value ranges,
// and only allow moving viewport on X axis.
func New(w, h int, opts ...Option) Model {
	min := time.Now()
	max := min.Add(time.Second)
	m := Model{
		Model: linechart.New(w, h, float64(min.Unix()), float64(max.Unix()), 0, 1,
			linechart.WithXYSteps(4, 2),
			linechart.WithXLabelFormatter(DateTimeLabelFormatter()),
			linechart.WithAutoXYRange(),                        // automatically adjust value ranges
			linechart.WithUpdateHandler(DateUpdateHandler(1))), // only scroll on X axis, increments by 1 day
		dLineStyle: runes.ArcLineStyle,
		dStyle:     lipgloss.NewStyle(),
		dSets:      make(map[string]*dataSet),
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.UpdateGraphSizes()
	m.rescaleData()
	if _, ok := m.dSets[DefaultDataSetName]; !ok {
		m.dSets[DefaultDataSetName] = m.newDataSet()
	}
	return m
}

// newDataSet returns a new initialize *dataSet.
func (m *Model) newDataSet() *dataSet {
	ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY()) // y scale factor
	sz := len(m.timeBuckets)
	// do not offset or scale X time values
	offset := canvas.Float64Point{X: 0, Y: m.ViewMinY()}
	scale := canvas.Float64Point{X: 1, Y: ys}
	return &dataSet{
		LineStyle: m.dLineStyle,
		Style:     m.dStyle,
		tBuf:      buffer.NewFloat64PointScaleBuffer(offset, scale),
		points:    make([][]float64, sz, sz),
	}
}

// resetTimeBuckets will reinitialize time chunks representing
// the start time of each graph column
func (m *Model) resetTimeBuckets() {
	// width of graphing area includes Y axis (used for line runes)
	width := m.Width() - m.Origin().X
	if len(m.timeBuckets) != width {
		m.timeBuckets = make([]time.Time, width, width)
	}
	// from min displayed time to max display time, divide the time range into chunks
	start := m.ViewMinX()
	end := m.ViewMaxX()
	rangeSz := end - start
	increment := rangeSz / float64(width)
	for i := 0; i < width; i++ {
		t := int64(start) + int64(math.Round(increment*float64(i)))
		m.timeBuckets[i] = time.Unix(t, 0)
	}
}

// rescaleData will reinitialize time chunks and
// map time points into graph columns for display
func (m *Model) rescaleData() {
	m.resetTimeBuckets()
	width := len(m.timeBuckets)
	if width < 1 {
		return
	}
	// rescale time points buffer and replace values into buckets
	ys := float64(m.Origin().Y) / (m.ViewMaxY() - m.ViewMinY()) // y scale factor
	offset := canvas.Float64Point{X: 0, Y: m.ViewMinY()}
	scale := canvas.Float64Point{X: 1, Y: ys}
	for _, ds := range m.dSets {
		if ds.tBuf.Offset() != offset {
			ds.tBuf.SetOffset(offset)
		}
		if ds.tBuf.Scale() != scale {
			ds.tBuf.SetScale(scale)
		}
		m.resetPoints(ds)
	}
}

// resetPoints will repopulate the dataset's points
// based on current view time range
func (m *Model) resetPoints(ds *dataSet) {
	// add all points to the graphing area
	// if there are points that are before
	// the graph, then need to use those points
	// otherwise the lines do not look continuous
	// and graph line starts or ends with Y value as 0
	width := len(m.timeBuckets)
	if width == 0 {
		return
	}
	ds.points = make([][]float64, width, width)
	var valBefore float64
	for _, v := range ds.tBuf.ReadAll() {
		rejBefore := m.addToPoints(ds.points, v)
		if rejBefore {
			valBefore = v.Y
		}
	}
	// add Y value to points from the beginning that is missing points
	idx := 0
	for len(ds.points[idx]) == 0 {
		ds.points[idx] = append(ds.points[idx], valBefore)
		idx++
		if idx == len(ds.points) {
			break
		}
	}
}

// addToPoints attempts to add given Float64Point containing
// seconds since epoch and Y value to points buckets.
// Returns whether point is rejected due to
// being less than the min view time.
func (m *Model) addToPoints(points [][]float64, f canvas.Float64Point) bool {
	// do nothing if point is out time slice range
	if f.X < m.ViewMinX() {
		return true
	} else if f.X > m.ViewMaxX() {
		return false
	}
	// check every bucket that the current point can fit
	t := int64(f.X)
	for idx, curTime := range m.timeBuckets {
		if t < curTime.Unix() {
			break
		}
		nextTime := curTime
		if (idx + 1) < len(m.timeBuckets) {
			nextTime = m.timeBuckets[idx+1]
		}
		if curTime == nextTime {
			if curTime.Unix() <= t {
				points[idx] = append(points[idx], f.Y)
			}
		} else { // assumes curTime < nextTime
			if (curTime.Unix() <= t) && (t < nextTime.Unix()) {
				points[idx] = append(points[idx], f.Y)
			}
		}
	}
	return false
}

// ClearAllData will reset stored data values in all data sets.
func (m *Model) ClearAllData() {
	for _, ds := range m.dSets {
		ds.tBuf.Clear()
	}
	m.dSets[DefaultDataSetName] = m.newDataSet()
}

// ClearDataSet will erase stored data set given by name string.
func (m *Model) ClearDataSet(n string) {
	if ds, ok := m.dSets[n]; ok {
		ds.tBuf.Clear()
	}
}

// SetTimeRange updates the minimum and maximum expected time values.
// Existing data will be rescaled.
func (m *Model) SetTimeRange(min, max time.Time) {
	m.Model.SetXRange(float64(min.Unix()), float64(max.Unix()))
	m.rescaleData()
}

// SetYRange updates the minimum and maximum expected Y values.
// Existing data will be rescaled.
func (m *Model) SetYRange(min, max float64) {
	m.Model.SetYRange(min, max)
	m.rescaleData()
}

// SetViewTimeRange updates the displayed minimum and maximum time values.
// Existing data will be rescaled.
func (m *Model) SetViewTimeRange(min, max time.Time) {
	m.Model.SetViewXRange(float64(min.Unix()), float64(max.Unix()))
	m.rescaleData()
}

// SetViewYRange updates the displayed minimum and maximum Y values.
// Existing data will be rescaled.
func (m *Model) SetViewYRange(min, max float64) {
	m.Model.SetViewYRange(min, max)
	m.rescaleData()
}

// SetViewTimeAndYRange updates the displayed minimum and maximum time and Y values.
// Existing data will be rescaled.
func (m *Model) SetViewTimeAndYRange(minX, maxX time.Time, minY, maxY float64) {
	m.Model.SetViewXRange(float64(minX.Unix()), float64(maxX.Unix()))
	m.Model.SetViewYRange(minY, maxY)
	m.rescaleData()
}

// Resize will change timeserieslinechart display width and height.
// Existing data will be rescaled.
func (m *Model) Resize(w, h int) {
	m.Model.Resize(w, h)
	m.rescaleData()
}

// SetStyles will set the default styles of data sets.
func (m *Model) SetStyles(ls runes.LineStyle, s lipgloss.Style) {
	m.dLineStyle = ls
	m.dStyle = s
	m.SetDataSetStyles(DefaultDataSetName, ls, s)
}

// SetDataSetStyles will set the styles of the given data set by name string.
func (m *Model) SetDataSetStyles(n string, ls runes.LineStyle, s lipgloss.Style) {
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	ds := m.dSets[n]
	ds.LineStyle = ls
	ds.Style = s
}

// Push will push a TimePoint data value to the default data set
// to be displayed with Draw.
func (m *Model) Push(t TimePoint) {
	m.PushDataSet(DefaultDataSetName, t)
}

// Push will push a TimePoint data value to a data set
// to be displayed with Draw. Using given data set by name string.
func (m *Model) PushDataSet(n string, t TimePoint) {
	f := canvas.Float64Point{X: float64(t.Time.Unix()), Y: t.Value}
	// auto adjust x and y ranges if enabled
	if m.AutoAdjustRange(f) {
		m.UpdateGraphSizes()
		m.rescaleData()
	}
	if _, ok := m.dSets[n]; !ok {
		m.dSets[n] = m.newDataSet()
	}
	ds := m.dSets[n]
	ds.tBuf.Push(f)
	m.resetPoints(ds)
}

// Draw will draw lines runes displayed from right to left
// of the graphing area of the canvas. Uses default data set.
func (m *Model) Draw() {
	m.DrawDataSets([]string{DefaultDataSetName})
}

// DrawAll will draw lines runes for all data sets from right
// to left of the graphing area of the canvas.
func (m *Model) DrawAll() {
	names := make([]string, 0, len(m.dSets))
	for n, ds := range m.dSets {
		if ds.tBuf.Length() > 0 {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	m.DrawDataSets(names)
}

// DrawDataSets will draw lines runes from right to left
// of the graphing area of the canvas for each data set given
// by name strings.
func (m *Model) DrawDataSets(names []string) {
	if len(names) == 0 {
		return
	}
	m.Clear()
	m.DrawXYAxisAndLabel()
	for _, n := range names {
		if ds, ok := m.dSets[n]; ok {
			var lastVal int
			l := make([]int, 0, len(ds.points))
			for _, bucket := range ds.points {
				// if no data for bucket, use previous value
				// so graph lines do not have any gaps
				if len(bucket) == 0 {
					l = append(l, lastVal)
					continue
				}
				// get average point value
				var sum float64
				for _, v := range bucket {
					sum += v
				}
				avg := sum / float64(len(bucket))
				lastVal = int(math.Round(avg))
				l = append(l, lastVal)
			}
			// convert to canvas coordinates and avoid drawing below X axis
			yCoords := canvas.CanvasYCoordinates(m.Origin().Y, l)
			if m.XStep() > 0 {
				for i, v := range yCoords {
					if v > m.Origin().Y {
						yCoords[i] = m.Origin().Y
					}
				}
			}
			startX := m.Canvas.Width() - len(yCoords)
			graph.DrawLineSequence(&m.Canvas,
				(startX == m.Origin().X),
				startX,
				yCoords,
				ds.LineStyle,
				ds.Style)
		}
	}
}

// Update processes bubbletea Msg to by invoking
// UpdateHandlerFunc callback if linechart is focused.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.Focused() {
		return m, nil
	}
	m.UpdateHandler(&m.Model, msg)
	m.rescaleData()
	return m, nil
}
