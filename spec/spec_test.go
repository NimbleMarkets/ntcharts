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
		YAxis: YAxis{Title: "day", Min: f64(0), Max: f64(6)},
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
		out.Data.Series[0].Values[0].Size == nil || *out.Data.Series[0].Values[0].Size != 4.5 {
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
