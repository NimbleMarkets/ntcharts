// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

// OHLCModel is a candlestick chart with a volume panel sharing its time scale,
// candle widths, colours and Y-axis gutter. The bottom quarter is reserved for
// volume; X labels appear only below that panel. Resize and Draw retain the
// spec data. Panning and zooming are not supported.
type OHLCModel struct {
	canvas.Model
	spec Spec
}

func buildOHLCVolume(s Spec) (*OHLCModel, error) {
	m := &OHLCModel{Model: canvas.New(s.Width, s.Height), spec: s}
	if err := m.draw(); err != nil {
		return nil, err
	}
	return m, nil
}

// Resize redraws both panels. Undersized windows are blank and recover on resize.
func (m *OHLCModel) Resize(w, h int) {
	w, h = max(0, w), max(0, h)
	m.Model.Resize(w, h)
	m.ViewWidth, m.ViewHeight = w, h
	m.Draw()
}

// Update handles window resizing without enabling independent panel pan/zoom.
func (m OHLCModel) Update(msg tea.Msg) (OHLCModel, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.Resize(size.Width, size.Height)
	}
	return m, nil
}

// Draw redraws the retained spec at the current canvas dimensions.
func (m *OHLCModel) Draw() { _ = m.draw() }

func (m *OHLCModel) draw() error {
	m.Clear()
	s := m.spec
	s.Width, s.Height = m.Width(), m.Height()
	if s.Height < 7 {
		return fmt.Errorf("spec: ohlc volume requires at least 7 rows")
	}
	rows := max(2, (s.Height-2)/4)
	s.Height -= rows
	maxV := 0.0
	for _, ser := range s.Data.Series {
		for _, p := range ser.OHLC {
			if p.V != nil {
				maxV = max(maxV, *p.V)
			}
		}
	}
	label := FormatValue(Format{Kind: "si"}, maxV)
	price, err := buildOHLCPrice(s, len(label))
	if err != nil {
		return err
	}
	// Preserve the price canvas exactly, moving its two X-axis rows below volume.
	top := price.Height() - 2
	for y := 0; y < price.Height(); y++ {
		dest := y
		if y >= top {
			dest += rows
		}
		for x := 0; x < m.Width(); x++ {
			m.SetCell(canvas.Point{X: x, Y: dest}, price.Canvas.Cell(canvas.Point{X: x, Y: y}))
		}
	}
	gutter := price.Origin().X
	// The relocated baseline belongs to volume, not the price minimum.
	for x := 0; x < gutter; x++ {
		m.SetCell(canvas.Point{X: x, Y: m.Height() - 2}, canvas.NewCell(' '))
	}
	if gutter > 0 {
		m.SetCell(canvas.Point{X: gutter - 1, Y: m.Height() - 2}, canvas.NewCellWithStyle('0', price.LabelStyle))
	}
	for y := top; y < top+rows; y++ {
		m.SetCell(canvas.Point{X: gutter, Y: y}, canvas.NewCellWithStyle('│', price.AxisStyle))
	}
	for i, r := range label {
		m.SetCell(canvas.Point{X: gutter - len(label) + i, Y: top}, canvas.NewCellWithStyle(r, price.LabelStyle))
	}
	pts := s.Data.Series[0].OHLC
	times := make([]time.Time, len(pts))
	for i, p := range pts {
		times[i], _ = PointTime(p.T)
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
	width := autoCandleWidth(price.GraphWidth(), times, tMin, tMax)
	scale := float64(price.GraphWidth()) / (price.ViewMaxX() - price.ViewMinX())
	for i, p := range pts {
		if p.V == nil || *p.V == 0 || maxV == 0 {
			continue
		}
		x := (float64(times[i].UnixMilli())/1e3 - price.ViewMinX()) * scale
		center := price.CandleColumn(x, width)
		style := candleStyle(s.Theme, 0, defaultUpColor)
		if p.C < p.O {
			style = candleStyle(s.Theme, 1, defaultDownColor)
		}
		height := max(1, int(math.Ceil(*p.V/maxV*float64(rows))))
		for col := center - (width-1)/2; col <= center+width/2; col++ {
			if col <= gutter || col >= price.GraphRight() {
				continue
			}
			for y := top + rows - height; y < top+rows; y++ {
				m.SetCell(canvas.Point{X: col, Y: y}, canvas.NewCellWithStyle('█', style))
			}
		}
	}
	return nil
}
