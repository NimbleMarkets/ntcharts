package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

func f64(v float64) *float64 { return &v }

func TestValidateSchemaV1(t *testing.T) {
	base := Spec{
		Type: ChartTypeBar, Width: 40, Height: 10,
		Data: Data{Series: []Series{{Name: "a", Values: []DataPoint{{Y: 1}}}}},
	}

	t.Run("valid base", func(t *testing.T) {
		if err := base.Validate(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("bad orientation", func(t *testing.T) {
		s := base
		s.Options.Orientation = "diagonal"
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "orientation") {
			t.Fatalf("expected orientation error, got %v", err)
		}
	})

	t.Run("bad format kind", func(t *testing.T) {
		s := base
		s.YAxis.Format = Format{Kind: "roman-numerals"}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "format") {
			t.Fatalf("expected format-kind error, got %v", err)
		}
	})

	t.Run("bad x_axis type", func(t *testing.T) {
		s := base
		s.XAxis.Type = "logarithmic"
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "x_axis type") {
			t.Fatalf("expected x_axis type error, got %v", err)
		}
	})

	t.Run("x_axis type accepts the three constants and empty", func(t *testing.T) {
		for _, typ := range []string{"", XAxisCategory, XAxisTime, XAxisValue} {
			s := base
			s.XAxis.Type = typ
			if err := s.Validate(); err != nil {
				t.Fatalf("x_axis type %q: expected valid, got %v", typ, err)
			}
		}
	})

	t.Run("bad axis scale", func(t *testing.T) {
		s := base
		s.XAxis.Scale = "symlog"
		if err := s.Validate(); err == nil || err.Error() != `spec: x_axis scale "symlog" must be linear or log` {
			t.Fatalf("expected x_axis scale error, got %v", err)
		}
		s = base
		s.YAxis.Scale = "Log"
		if err := s.Validate(); err == nil || err.Error() != `spec: y_axis scale "Log" must be linear or log` {
			t.Fatalf("expected y_axis scale error, got %v", err)
		}
	})

	t.Run("axis scale accepts empty, linear and log", func(t *testing.T) {
		for _, scale := range []string{"", ScaleLinear, ScaleLog} {
			s := base
			s.XAxis.Scale, s.YAxis.Scale = scale, scale
			if err := s.Validate(); err != nil {
				t.Fatalf("scale %q: expected valid, got %v", scale, err)
			}
		}
	})

	t.Run("bad series type", func(t *testing.T) {
		s := base
		s.Data.Series = []Series{{Name: "a", Type: "pie", Values: []DataPoint{{Y: 1}}}}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "series[0] type") {
			t.Fatalf("expected series type error, got %v", err)
		}
	})

	t.Run("series type accepts empty and any known chart type", func(t *testing.T) {
		for _, typ := range []string{"", "bar", "line"} {
			s := base
			s.Data.Series = []Series{{Name: "a", Type: typ, Values: []DataPoint{{Y: 1}}}}
			if err := s.Validate(); err != nil {
				t.Fatalf("series type %q: expected valid, got %v", typ, err)
			}
		}
	})

	t.Run("heatmap requires heat data", func(t *testing.T) {
		s := Spec{Type: ChartTypeHeatmap, Width: 40, Height: 10}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "Heat") {
			t.Fatalf("expected Heat-required error, got %v", err)
		}
	})

	t.Run("heatmap with cells needs no series", func(t *testing.T) {
		s := Spec{Type: ChartTypeHeatmap, Width: 40, Height: 10,
			Heat: &HeatData{Cells: []HeatCell{{X: 0, Y: 0, Z: 1}}}}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("ohlc requires ohlc points", func(t *testing.T) {
		s := Spec{Type: ChartTypeOHLC, Width: 40, Height: 10,
			Data: Data{Series: []Series{{Name: "px", Values: []DataPoint{{Y: 1}}}}}}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "OHLC") {
			t.Fatalf("expected OHLC-required error, got %v", err)
		}
	})
}

func TestSpecJSONRoundTrip(t *testing.T) {
	in := Spec{
		Type: ChartTypeHeatmap, Title: "t", Width: 30, Height: 8,
		XAxis: XAxis{Title: "hour", Type: XAxisValue, Format: Format{Kind: "number", Precision: f64i(0)}},
		YAxis: YAxis{Title: "day", Min: f64(0), Max: f64(6), Labels: []string{"Mon", "Tue"}},
		Data: Data{Series: []Series{{Name: "s", Values: []DataPoint{
			{X: 1.0, Y: 2.5, Size: f64(4.5)},
		}}}},
		Heat: &HeatData{
			Cells:    []HeatCell{{X: 0, Y: 1, Z: 3.5}},
			MinValue: f64(0), MaxValue: f64(10),
		},
		Theme: Theme{Gradient: []string{"#000000", "#ff0000"}},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Spec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.XAxis.Title != "hour" || out.YAxis.Max == nil || *out.YAxis.Max != 6 ||
		out.Heat == nil || len(out.Heat.Cells) != 1 || out.Heat.Cells[0].Z != 3.5 ||
		len(out.Theme.Gradient) != 2 ||
		len(out.Data.Series) != 1 || len(out.Data.Series[0].Values) != 1 ||
		out.Data.Series[0].Values[0].Size == nil || *out.Data.Series[0].Values[0].Size != 4.5 ||
		len(out.YAxis.Labels) != 2 || out.YAxis.Labels[0] != "Mon" || out.YAxis.Labels[1] != "Tue" {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

// TestSpecJSONMinimalOmitsZeroStructs verifies the omitzero tags on
// Spec.XAxis / Spec.YAxis / Spec.Options / Spec.Theme (and Format on each
// axis) actually suppress emitting empty "{}" objects for a minimal Spec,
// unlike the no-op omitempty tags they replaced.
func TestSpecJSONMinimalOmitsZeroStructs(t *testing.T) {
	in := Spec{
		Type: ChartTypeBar, Width: 10, Height: 5,
		Data: Data{Series: []Series{{Name: "a", Values: []DataPoint{{Y: 1}}}}},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, key := range []string{`"x_axis"`, `"y_axis"`, `"options"`, `"theme"`} {
		if strings.Contains(got, key) {
			t.Fatalf("minimal Spec JSON unexpectedly contains %s: %s", key, got)
		}
	}
}

func f64i(v int) *int { return &v }

func TestFormatIsZero(t *testing.T) {
	if !(Format{}).IsZero() {
		t.Fatal("zero Format should report IsZero")
	}
	if (Format{Kind: "si"}).IsZero() {
		t.Fatal("non-zero Format must not report IsZero")
	}
}

func TestSpecJSONScaleRoundTrip(t *testing.T) {
	in := Spec{
		Type: ChartTypeScatter, Width: 40, Height: 10,
		XAxis: XAxis{Scale: ScaleLog},
		YAxis: YAxis{Scale: ScaleLog},
		Data:  Data{Series: []Series{{Name: "a", Values: []DataPoint{{X: 1.0, Y: 1}}}}},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); !strings.Contains(got, `"x_axis":{"scale":"log"}`) || !strings.Contains(got, `"y_axis":{"scale":"log"}`) {
		t.Fatalf("scale missing from JSON: %s", got)
	}
	var out Spec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.XAxis.Scale != ScaleLog || out.YAxis.Scale != ScaleLog {
		t.Fatalf("round-trip mismatch: %+v / %+v", out.XAxis, out.YAxis)
	}

	// An empty scale is omitted, so linear specs serialize exactly as before.
	in.XAxis.Scale, in.YAxis.Scale = "", ""
	in.YAxis.Min = f64(1)
	if b, err = json.Marshal(in); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "scale") {
		t.Fatalf("empty scale must be omitted: %s", b)
	}
}

func TestY2AxisJSONRoundTrip(t *testing.T) {
	base := Spec{
		Type: ChartTypeTimeSeries, Width: 40, Height: 10,
		Data: Data{Series: []Series{{Name: "a", Values: []DataPoint{{Y: 1}}}}},
	}
	t.Run("omitted", func(t *testing.T) {
		b, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "y2_axis") || strings.Contains(string(b), `"y_axis":"`) {
			t.Fatalf("nil Y2Axis / empty Series.YAxis must be omitted:\n%s", b)
		}
	})
	t.Run("preserved", func(t *testing.T) {
		s := base
		s.Y2Axis = &YAxis{Min: f64(5), Max: f64(20), Format: Format{Kind: "si"}, Scale: ScaleLinear}
		s.Data.Series = append(s.Data.Series, Series{Name: "r", YAxis: YAxisRight, Values: []DataPoint{{Y: 7}}})
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		var back Spec
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatal(err)
		}
		if back.Y2Axis == nil || *back.Y2Axis.Min != 5 || *back.Y2Axis.Max != 20 ||
			back.Y2Axis.Format.Kind != "si" || back.Y2Axis.Scale != ScaleLinear {
			t.Fatalf("y2_axis did not round-trip: %+v", back.Y2Axis)
		}
		if back.Data.Series[1].YAxis != YAxisRight || back.Data.Series[0].YAxis != "" {
			t.Fatalf("series y_axis did not round-trip: %q, %q", back.Data.Series[0].YAxis, back.Data.Series[1].YAxis)
		}
		if !strings.Contains(string(b), `"y2_axis":{`) || !strings.Contains(string(b), `"y_axis":"right"`) {
			t.Fatalf("json keys:\n%s", b)
		}
	})
}

func TestValidateY2Axis(t *testing.T) {
	base := Spec{
		Type: ChartTypeTimeSeries, Width: 40, Height: 10,
		Data: Data{Series: []Series{{Name: "a", Values: []DataPoint{{Y: 1}}}}},
	}
	t.Run("bad series y_axis", func(t *testing.T) {
		s := base
		s.Data.Series = []Series{{Name: "a", YAxis: "middle", Values: []DataPoint{{Y: 1}}}}
		if err := s.Validate(); err == nil || err.Error() != `spec: series[0] y_axis "middle" must be left or right` {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("right requires y2_axis", func(t *testing.T) {
		s := base
		s.Data.Series = []Series{{Name: "a", YAxis: YAxisRight, Values: []DataPoint{{Y: 1}}}}
		if err := s.Validate(); err == nil || err.Error() != `spec: series[0] y_axis "right" requires y2_axis` {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("left and empty are valid; y2_axis without right series is valid", func(t *testing.T) {
		s := base
		s.Y2Axis = &YAxis{}
		s.Data.Series = []Series{
			{Name: "a", YAxis: YAxisLeft, Values: []DataPoint{{Y: 1}}},
			{Name: "b", Values: []DataPoint{{Y: 1}}},
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})
	t.Run("right with y2_axis is valid", func(t *testing.T) {
		s := base
		s.Y2Axis = &YAxis{}
		s.Data.Series = []Series{{Name: "a", YAxis: YAxisRight, Values: []DataPoint{{Y: 1}}}}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})
	t.Run("bad y2_axis format", func(t *testing.T) {
		s := base
		s.Y2Axis = &YAxis{Format: Format{Kind: "roman-numerals"}}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "y2_axis format kind") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("bad y2_axis scale", func(t *testing.T) {
		s := base
		s.Y2Axis = &YAxis{Scale: "symlog"}
		if err := s.Validate(); err == nil || err.Error() != `spec: y2_axis scale "symlog" must be linear or log` {
			t.Fatalf("got %v", err)
		}
	})
}

func TestValidateRejectsSameNameOnDifferentYAxes(t *testing.T) {
	s := Spec{
		Type: ChartTypeTimeSeries, Width: 40, Height: 10,
		Y2Axis: &YAxis{},
		Data: Data{Series: []Series{
			{Name: "a", YAxis: YAxisRight, Values: []DataPoint{{Y: 100}}},
			{Name: "b", Values: []DataPoint{{Y: 1}}},
			{Name: "a", Values: []DataPoint{{Y: 2}}},
		}},
	}
	if err := s.Validate(); err == nil || err.Error() != `spec: series[2] "a" is also series[0] but on a different y_axis` {
		t.Fatalf("got %v", err)
	}
	// the same name on the same axis is still allowed (it merges)
	s.Data.Series[2].YAxis = YAxisRight
	if err := s.Validate(); err != nil {
		t.Fatalf("same axis: expected valid, got %v", err)
	}
}
