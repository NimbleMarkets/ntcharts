// ntcharts - Copyright (c) 2026 Neomantra Corp.

package echarts

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

func FuzzToECharts(f *testing.F) {
	seeds := []string{
		`{"type":"bar","width":400,"height":300,"data":{"series":[{"name":"s1","values":[{"x":"A","y":10},{"x":"B","y":-5}]}]}}`,
		`{"type":"bar","width":400,"height":300,"y2_axis":{"min":0,"max":100},"data":{"series":[{"name":"rev","type":"bar","values":[{"x":"Q1","y":20},{"x":"Q2","y":30}]},{"name":"growth","type":"line","y_axis":"right","values":[{"x":"Q1","y":15},{"x":"Q2","y":45}]}]}}`,
		`{"type":"line","width":400,"height":300,"x_axis":{"scale":"log"},"y_axis":{"scale":"log"},"data":{"series":[{"name":"s1","values":[{"x":1,"y":1},{"x":10,"y":100}]}]}}`,
		`{"type":"scatter","width":400,"height":300,"data":{"series":[{"name":"pts","values":[{"x":1,"y":2,"size":4},{"x":3,"y":4}]}]}}`,
		`{"type":"timeseries","width":400,"height":300,"data":{"series":[{"name":"ts","values":[{"x":"2026-01-01T00:00:00Z","y":5},{"x":"2026-01-02T00:00:00Z","y":8}]}]}}`,
		`{"type":"sparkline","width":200,"height":80,"data":{"series":[{"name":"sp","values":[{"y":1},{"y":4},{"y":2}]}]}}`,
		`{"type":"heatmap","width":400,"height":300,"heat":{"cells":[{"x":0,"y":0,"z":1},{"x":1,"y":1,"z":5}]},"theme":{"gradient":["#000000","#ffffff"]}}`,
		`{"type":"ohlc","width":500,"height":350,"options":{"volume":true,"candle_style":"block"},"data":{"series":[{"name":"btc","ohlc":[{"t":"2026-01-01T00:00:00Z","o":100,"h":110,"l":95,"c":105,"v":500},{"t":"2026-01-02T00:00:00Z","o":105,"h":115,"l":100,"c":102,"v":300}]}]}}`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var s spec.Spec
		if err := json.Unmarshal(data, &s); err != nil {
			return
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
		ch, err := ToECharts(s)
		if valErr != nil && err == nil {
			t.Fatalf("ToECharts succeeded when Validate failed: %v", valErr)
		}
		if err == nil {
			if ch == nil {
				t.Fatalf("ToECharts returned nil without error")
			}
			if r, ok := ch.(interface{ Render(io.Writer) error }); ok {
				var buf bytes.Buffer
				if rErr := r.Render(&buf); rErr != nil {
					t.Fatalf("ToECharts Render failed: %v", rErr)
				}
			}
		}
	})
}
