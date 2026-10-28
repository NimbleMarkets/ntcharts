// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

func TestOHLCVolumeValidation(t *testing.T) {
	for _, v := range []float64{-1, math.NaN(), math.Inf(1)} {
		s := ohlcSpec()
		s.Data.Series[0].OHLC[0].V = &v
		if err := s.Validate(); err == nil || !strings.HasPrefix(err.Error(), "spec:") {
			t.Fatalf("volume %v: %v", v, err)
		}
	}
	s := ohlcSpec()
	s.Type = ChartTypeLine
	s.Options.Volume = true
	if s.Validate() == nil {
		t.Fatal("volume on line accepted")
	}
}

func TestOHLCVolumeOffGolden(t *testing.T) {
	s := ohlcSpec()
	a, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	view := stripAnsi(a.(*timeserieslinechart.Model).View())
	golden(t, "ohlc_no_volume", view)
	before, _ := json.Marshal(s)
	if strings.Contains(string(before), `"v":`) || strings.Contains(string(before), `"volume":`) {
		t.Fatal("optional fields serialized")
	}
	s.Options.Volume = true // no V: identical model and output
	b, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if b.(*timeserieslinechart.Model).View() != a.(*timeserieslinechart.Model).View() {
		t.Fatal("empty volume changed output")
	}
	s.Options.Volume = false
	v := 123.0
	s.Data.Series[0].OHLC[0].V = &v
	b, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if b.(*timeserieslinechart.Model).View() != a.(*timeserieslinechart.Model).View() {
		t.Fatal("disabled volume changed output")
	}
	encoded, _ := json.Marshal(s)
	var decoded Spec
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if *decoded.Data.Series[0].OHLC[0].V != v {
		t.Fatal("volume lost in JSON")
	}
}

func TestOHLCVolumeAlignment(t *testing.T) {
	for _, count := range []int{3, 120} {
		for _, w := range []int{32, 80, 120} {
			for _, gaps := range []bool{false, true} {
				s := ohlcSpec()
				s.Width = w
				s.Height = 24
				s.Options.Volume = true
				pts := make([]OHLCPoint, count)
				start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				for i := range pts {
					day := i
					if gaps && i >= count/2 {
						day += 7
					}
					v := 12000.0
					pts[i] = OHLCPoint{T: start.Add(time.Duration(day) * 24 * time.Hour), O: 102, H: 110, L: 100, C: 108, V: &v}
				}
				s.Data.Series[0].OHLC = pts
				built, err := Build(s)
				if err != nil {
					t.Fatal(err)
				}
				m := built.(*OHLCModel)
				// Collect the columns actually painted by candle bodies, independently of
				// the volume renderer's coordinates. Both use identical width and clipping.
				top := s.Height - 2 - max(2, (s.Height-2)/4)
				candleCols := map[int]bool{}
				volumeCols := map[int]bool{}
				for y := 0; y < top; y++ {
					for x := 0; x < w; x++ {
						c := m.Cell(canvas.Point{X: x, Y: y})
						if strings.ContainsRune("┃╽╿", c.Rune) {
							candleCols[x] = true
						}
					}
				}
				for x := 0; x < w; x++ {
					if m.Cell(canvas.Point{X: x, Y: s.Height - 3}).Rune == '█' {
						volumeCols[x] = true
					}
				}
				if len(candleCols) == 0 {
					t.Fatal("no candles")
				}
				// Compare the full body spans, then check the independent top wick
				// positions against the centers of separated sparse bars.
				if len(candleCols) != len(volumeCols) {
					t.Fatalf("width %d gaps %v: candle/bar spans differ %d/%d", w, gaps, len(candleCols), len(volumeCols))
				}
				for x := range candleCols {
					if !volumeCols[x] {
						t.Fatalf("candle column %d has no bar", x)
					}
				}
				if count == 3 {
					// Isolate each volume bar so edge-inset candles that touch
					// or overlap still have independently testable centers.
					for i := range pts {
						isolated := s
						isolated.Data.Series = []Series{{Name: "px", OHLC: append([]OHLCPoint(nil), pts...)}}
						for j := range pts {
							if j != i {
								isolated.Data.Series[0].OHLC[j].V = nil
							}
						}
						one, err := Build(isolated)
						if err != nil {
							t.Fatal(err)
						}
						panel := one.(*OHLCModel)
						lo, hi := w, -1
						for x := 0; x < w; x++ {
							if panel.Cell(canvas.Point{X: x, Y: s.Height - 3}).Rune == '█' {
								lo = min(lo, x)
								hi = max(hi, x)
							}
						}
						if hi < lo {
							t.Fatal("missing isolated volume")
						}
						if !gaps && hi-lo+1 != 7 {
							t.Fatalf("width %d: bar width %d, want 7", w, hi-lo+1)
						}
						wick := panel.Cell(canvas.Point{X: (lo + hi) / 2, Y: 0})
						if wick.Rune != '│' || !reflect.DeepEqual(wick.Style.GetForeground(), candleStyle(s.Theme, 0, defaultUpColor).GetForeground()) {
							t.Fatalf("width %d gaps %v: bar center %d misses wick", w, gaps, (lo+hi)/2)
						}
					}
				}
				original := m.View()
				m.Resize(1, 1)
				m.Resize(w, s.Height)
				if m.View() != original {
					t.Fatal("resize did not restore both panels")
				}
			}
		}
	}
}

func TestOHLCMissingAndZeroVolume(t *testing.T) {
	s := ohlcSpec()
	s.Options.Volume = true
	zero := 0.0
	s.Data.Series[0].OHLC[0].V = &zero
	b, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.(*OHLCModel).View(), "█") {
		t.Fatal("zero or missing volume drew a bar")
	}
	s.Height = 6
	if _, err := Build(s); err == nil {
		t.Fatal("undersized panel accepted")
	}
}

func TestOHLCVolumeLabelsAndColours(t *testing.T) {
	s := ohlcSpec()
	s.Width = 80
	s.Height = 24
	s.Options.Volume = true
	s.Theme.Palette = []string{"#123456", "#abcdef"}
	s.XAxis.Format = Format{Kind: "time", Layout: "Jan 02"}
	for i := range s.Data.Series[0].OHLC {
		v := 12300.0
		s.Data.Series[0].OHLC[i].V = &v
	}
	for _, style := range []string{CandleStyleLine, CandleStyleBlock} {
		s.Options.CandleStyle = style
		built, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		m := built.(*OHLCModel)
		lines := strings.Split(stripAnsi(m.View()), "\n")
		for _, line := range lines[:len(lines)-1] {
			if strings.Contains(line, "Jan") {
				t.Fatal("X label above volume")
			}
		}
		if !strings.Contains(lines[len(lines)-1], "Jan") || !strings.Contains(m.View(), "12.3k") {
			t.Fatal("missing time/SI labels")
		}
		// The four separated volume runs must follow candle direction, including
		// the down candle at index two, with exactly the requested palette styles.
		index := 0
		inBar := false
		for x := 0; x < s.Width; x++ {
			cell := m.Cell(canvas.Point{X: x, Y: s.Height - 3})
			if cell.Rune != '█' {
				inBar = false
				continue
			}
			if inBar {
				continue
			}
			inBar = true
			slot := 0
			if index == 2 {
				slot = 1
			}
			if !reflect.DeepEqual(cell.Style.GetForeground(), candleStyle(s.Theme, slot, "").GetForeground()) {
				t.Fatal("bar style differs from candle direction")
			}
			index++
		}
		if index != 4 {
			t.Fatalf("got %d volume runs", index)
		}
	}
}

func TestOHLCVolumeResizeViewport(t *testing.T) {
	for _, viaUpdate := range []bool{false, true} {
		s := ohlcSpec()
		s.Width, s.Height = 40, 16
		s.Options.Volume = true
		s.XAxis.Format = Format{Kind: "time", Layout: "Jan 02"}
		v := 12300.0
		for i := range s.Data.Series[0].OHLC {
			s.Data.Series[0].OHLC[i].V = &v
		}
		built, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		m := built.(*OHLCModel)
		for _, size := range [][2]int{{60, 30}, {32, 12}, {80, 24}} {
			w, h := size[0], size[1]
			if viaUpdate {
				updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
				m = &updated
			} else {
				m.Resize(w, h)
			}
			lines := strings.Split(stripAnsi(m.View()), "\n")
			if len(lines) != h {
				t.Fatalf("Update=%v: got %d rows, want %d", viaUpdate, len(lines), h)
			}
			for _, line := range lines {
				if len([]rune(line)) != w {
					t.Fatalf("Update=%v: got %d columns, want %d", viaUpdate, len([]rune(line)), w)
				}
			}
			if !strings.Contains(lines[h-3], "█") || !strings.Contains(lines[h-1], "Jan") {
				t.Fatal("resized view clips volume or time labels")
			}
			s.Width, s.Height = w, h
			fresh, err := Build(s)
			if err != nil {
				t.Fatal(err)
			}
			if m.View() != fresh.(*OHLCModel).View() {
				t.Fatal("resized view differs from fresh build")
			}
		}
	}
}
