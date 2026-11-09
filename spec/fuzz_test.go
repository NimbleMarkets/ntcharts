// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"encoding/json"
	"math"
	"testing"
	"unicode/utf8"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func FuzzSpecValidateAndBuild(f *testing.F) {
	seeds := []string{
		`{"type":"bar","width":30,"height":10,"data":{"series":[{"name":"s1","values":[{"x":"A","y":10},{"x":"B","y":-5}]}]}}`,
		`{"type":"bar","width":36,"height":12,"y2_axis":{"min":0,"max":100},"data":{"series":[{"name":"rev","type":"bar","values":[{"x":"Q1","y":20},{"x":"Q2","y":30}]},{"name":"growth","type":"line","y_axis":"right","values":[{"x":"Q1","y":15},{"x":"Q2","y":45}]}]}}`,
		`{"type":"line","width":30,"height":10,"x_axis":{"scale":"log"},"y_axis":{"scale":"log"},"data":{"series":[{"name":"s1","values":[{"x":1,"y":1},{"x":10,"y":100}]}]}}`,
		`{"type":"scatter","width":24,"height":8,"data":{"series":[{"name":"pts","values":[{"x":1,"y":2,"mark":{"id":"p1"}},{"x":3,"y":4}]}]}}`,
		`{"type":"timeseries","width":32,"height":10,"data":{"series":[{"name":"ts","values":[{"x":"2026-01-01T00:00:00Z","y":5},{"x":"2026-01-02T00:00:00Z","y":8}]}]}}`,
		`{"type":"sparkline","width":20,"height":5,"data":{"series":[{"name":"sp","values":[{"y":1},{"y":4},{"y":2}]}]}}`,
		`{"type":"heatmap","width":24,"height":8,"heat":{"cells":[{"x":0,"y":0,"z":1,"mark":{"id":"h1"}},{"x":1,"y":1,"z":5}]},"theme":{"gradient":["#000000","#ffffff"]}}`,
		`{"type":"ohlc","width":36,"height":14,"options":{"volume":true,"candle_style":"block"},"data":{"series":[{"name":"btc","ohlc":[{"t":"2026-01-01T00:00:00Z","o":100,"h":110,"l":95,"c":105,"v":500},{"t":"2026-01-02T00:00:00Z","o":105,"h":115,"l":100,"c":102,"v":300}]}]}}`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var s Spec
		if err := json.Unmarshal(data, &s); err != nil {
			return
		}
		// Bound dimensions and series sizes so fuzzing stays fast and memory-bounded.
		if s.Width > 80 {
			s.Width = 80
		}
		if s.Height > 40 {
			s.Height = 40
		}
		if len(s.Data.Series) > 8 {
			s.Data.Series = s.Data.Series[:8]
		}
		for i := range s.Data.Series {
			if len(s.Data.Series[i].Values) > 32 {
				s.Data.Series[i].Values = s.Data.Series[i].Values[:32]
			}
			if len(s.Data.Series[i].OHLC) > 32 {
				s.Data.Series[i].OHLC = s.Data.Series[i].OHLC[:32]
			}
		}
		if s.Heat != nil {
			if len(s.Heat.Cells) > 64 {
				s.Heat.Cells = s.Heat.Cells[:64]
			}
			if len(s.Heat.Matrix) > 16 {
				s.Heat.Matrix = s.Heat.Matrix[:16]
			}
			for i := range s.Heat.Matrix {
				if len(s.Heat.Matrix[i]) > 16 {
					s.Heat.Matrix[i] = s.Heat.Matrix[i][:16]
				}
			}
		}

		valErr := s.Validate()
		m, buildErr := Build(s)
		if valErr != nil && buildErr == nil {
			t.Fatalf("Build succeeded when Validate failed: %v", valErr)
		}
		if buildErr == nil && m != nil {
			if viewer, ok := m.(interface{ View() string }); ok {
				if !utf8.ValidString(viewer.View()) {
					t.Fatalf("Build View produced invalid UTF-8")
				}
			}
		}

		im, iErr := BuildInteractive(s)
		if valErr != nil && iErr == nil {
			t.Fatalf("BuildInteractive succeeded when Validate failed: %v", valErr)
		}
		if iErr == nil && im != nil {
			marks := im.Marks()
			if len(marks) > 0 {
				_, _ = im.Inspect(marks[0].ID)
				_, _ = im.Highlight(marks[0].ID)
				_ = im.Clear()
			}
			_ = im.HitTest(canvas.Point{X: s.Width / 2, Y: s.Height / 2})
			_ = im.Resize(max(1, s.Width/2), max(1, s.Height/2))
			if !utf8.ValidString(im.View()) {
				t.Fatalf("BuildInteractive View produced invalid UTF-8")
			}
		}
	})
}

func FuzzFormatValue(f *testing.F) {
	f.Add("number", int8(2), "$", "2006-01-02", 1234.567)
	f.Add("percent", int8(1), "", "", 0.425)
	f.Add("currency", int8(2), "€", "", -99.95)
	f.Add("si", int8(1), "", "", 1500000.0)
	f.Add("time", int8(-1), "", "15:04:05", 1700000000000.0)
	f.Add("si", int8(4), "", "", math.NaN())
	f.Add("number", int8(100), "", "", math.Inf(1))

	f.Fuzz(func(t *testing.T, kind string, precRaw int8, currency, layout string, v float64) {
		if !utf8.ValidString(kind) || !utf8.ValidString(currency) || !utf8.ValidString(layout) {
			return
		}
		prec := int(precRaw)
		fmtSpec := Format{
			Kind:      kind,
			Precision: &prec,
			Currency:  currency,
			Layout:    layout,
		}
		out := FormatValue(fmtSpec, v)
		if !utf8.ValidString(out) {
			t.Fatalf("FormatValue produced invalid UTF-8: %q", out)
		}
		logOut := formatLogValue(v)
		if !utf8.ValidString(logOut) {
			t.Fatalf("formatLogValue produced invalid UTF-8: %q", logOut)
		}
	})
}

func FuzzGradientAndHelpers(f *testing.F) {
	f.Add("#000000", "#ff8800", "#ffffff", "2026-03-15T12:00:00Z", 1700000000000.0)
	f.Add("#123456", "bad", "#abcdef", "2026-03-15", 0.0)

	f.Fuzz(func(t *testing.T, c1, c2, c3, timeStr string, numVal float64) {
		scale, err := gradientScale([]string{c1, c2, c3})
		if err == nil && len(scale) == 0 {
			t.Fatalf("gradientScale returned empty slice without error")
		}
		_, _ = PointTime(timeStr)
		_, _ = PointTime(numVal)
	})
}
