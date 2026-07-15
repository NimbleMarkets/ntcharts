package spec

import "testing"

func TestFormatValue(t *testing.T) {
	p := func(v int) *int { return &v }
	cases := []struct {
		name string
		f    Format
		v    float64
		want string
	}{
		{"default plain", Format{}, 12.5, "12.5"},
		{"number precision", Format{Kind: "number", Precision: p(2)}, 3.14159, "3.14"},
		{"percent default", Format{Kind: "percent"}, 0.42, "42%"},
		{"percent precision", Format{Kind: "percent", Precision: p(1)}, 0.4567, "45.7%"},
		{"currency default", Format{Kind: "currency"}, 1250, "$1250.00"},
		{"currency symbol", Format{Kind: "currency", Currency: "€", Precision: p(0)}, 99.9, "€100"},
		{"si thousands", Format{Kind: "si"}, 12500, "12.5k"},
		{"si millions", Format{Kind: "si", Precision: p(2)}, 3400000, "3.40M"},
		{"si small passthrough", Format{Kind: "si"}, 999, "999.0"},
		{"si negative", Format{Kind: "si"}, -2500000000, "-2.5G"},
		{"time layout", Format{Kind: "time", Layout: "2006-01"}, 1767225600000, "2026-01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FormatValue(c.f, c.v); got != c.want {
				t.Fatalf("FormatValue(%+v, %v) = %q, want %q", c.f, c.v, got, c.want)
			}
		})
	}
}

func TestLabelFormatterNilWhenZero(t *testing.T) {
	if (Format{}).labelFormatter() != nil {
		t.Fatal("zero Format must produce nil LabelFormatter (chart keeps its default)")
	}
	lf := Format{Kind: "si"}.labelFormatter()
	if lf == nil {
		t.Fatal("non-zero Format must produce a formatter")
	}
	if got := lf(0, 2000); got != "2.0k" {
		t.Fatalf("formatter(2000) = %q, want 2.0k", got)
	}
}
