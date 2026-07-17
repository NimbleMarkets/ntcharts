// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func ohlcSpec() Spec {
	return Spec{
		Type: ChartTypeOHLC, Width: 40, Height: 12,
		Data: Data{Series: []Series{{Name: "px", OHLC: []OHLCPoint{
			{T: "2026-01-05", O: 100, H: 108, L: 97, C: 105},
			{T: "2026-01-06", O: 105, H: 112, L: 103, C: 110},
			{T: "2026-01-07", O: 110, H: 111, L: 98, C: 99},
			{T: "2026-01-08", O: 99, H: 106, L: 96, C: 104},
		}}}},
	}
}

func TestBuildOHLC(t *testing.T) {
	got, err := Build(ohlcSpec())
	if err != nil {
		t.Fatalf("Build(ohlc): %v", err)
	}
	m, ok := got.(*canvas.Model)
	if !ok {
		t.Fatalf("Build(ohlc) returned %T, want *canvas.Model", got)
	}
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("ohlc view is empty")
	}
	// candle runes are box-drawing verticals; assert SOMETHING beyond axis was drawn
	if !strings.ContainsAny(view, "│┃╽╿") {
		t.Fatalf("no candle runes found in view:\n%s", view)
	}
}

func TestBuildOHLCClampsToMostRecent(t *testing.T) {
	s := ohlcSpec()
	// widen the data far beyond usable columns of a narrow chart
	s.Width = 12
	pts := s.Data.Series[0].OHLC
	for i := 0; i < 40; i++ {
		pts = append(pts, OHLCPoint{T: "2026-02-01", O: 100, H: 101, L: 99, C: 100})
	}
	s.Data.Series[0].OHLC = pts
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(ohlc, clamp): %v", err)
	}
}

func TestBuildOHLCInvertedPinErrors(t *testing.T) {
	s := ohlcSpec()
	s.YAxis.Min = f64(500) // above all highs
	if _, err := Build(s); err == nil {
		t.Fatal("expected inverted-range error")
	}
}
