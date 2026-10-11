// ntcharts - Copyright (c) 2026 Neomantra Corp.

package echarts

import (
	"encoding/json"
	"fmt"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func toEChartsCombo(s spec.Spec) (*charts.Bar, error) {
	c, err := spec.ResolveCategoricalCombo(s)
	if err != nil {
		return nil, err
	}
	left, right := yAxisOpt(s.YAxis, "left"), yAxisOpt(*s.Y2Axis, "right")
	left.Min, left.Max, right.Min, right.Max = c.Min, c.Max, c.Min2, c.Max2
	left.AxisLabel, err = comboAxisLabel(s.YAxis.Format, spec.ComboSeriesColor(s, c.BarIndex))
	if err != nil {
		return nil, err
	}
	right.AxisLabel, err = comboAxisLabel(s.Y2Axis.Format, spec.ComboSeriesColor(s, c.LineIndex))
	if err != nil {
		return nil, err
	}
	bar := charts.NewBar()
	bar.SetGlobalOptions(titleOpt(s), initOpt(s), legendOpt(s), gridOpt(s), charts.WithYAxisOpts(left), charts.WithXAxisOpts(opts.XAxis{Type: spec.XAxisCategory, Name: s.XAxis.Title}))
	bar.ExtendYAxis(right)
	bar.SetXAxis(c.Labels)
	bars := make([]opts.BarData, len(c.Labels))
	lines := make([]opts.LineData, len(c.Labels))
	for i := range c.Labels {
		// A typed nil pointer inside Value serializes as JSON null rather than
		// omitting the field, preserving the category slot on the web.
		bars[i] = opts.BarData{Value: c.Bars[i]}
		lines[i] = opts.LineData{Value: c.Line[i]}
	}
	bar.AddSeries(s.Data.Series[c.BarIndex].Name, bars, charts.WithItemStyleOpts(opts.ItemStyle{Color: spec.ComboSeriesColor(s, c.BarIndex)}))
	line := charts.NewLine()
	line.SetXAxis(c.Labels)
	color := spec.ComboSeriesColor(s, c.LineIndex)
	line.AddSeries(s.Data.Series[c.LineIndex].Name, lines,
		charts.WithLineChartOpts(opts.LineChart{YAxisIndex: 1, ConnectNulls: opts.Bool(false), ShowSymbol: opts.Bool(true)}),
		charts.WithLineStyleOpts(opts.LineStyle{Color: color}), charts.WithItemStyleOpts(opts.ItemStyle{Color: color}))
	// ECharts' line series has a higher default z than bars. Add it last too.
	bar.Overlap(line)
	return bar, nil
}

func comboAxisLabel(f spec.Format, color string) (*opts.AxisLabel, error) {
	label := &opts.AxisLabel{Show: opts.Bool(true), Color: color}
	if f.IsZero() {
		return label, nil
	}
	precision := -1
	if f.Precision != nil {
		precision = *f.Precision
	}
	if precision > 100 {
		return nil, fmt.Errorf("echarts: combo axis precision must be <= 100")
	}
	if f.Kind == "time" {
		return nil, fmt.Errorf("echarts: combo numeric Y axes do not support time formatting")
	}
	number := func(expr string, p int) string {
		if p < 0 {
			return "String(" + expr + ")"
		}
		return fmt.Sprintf("(%s).toFixed(%d)", expr, p)
	}
	expr := number("v", precision)
	switch f.Kind {
	case "percent":
		if precision < 0 {
			precision = 0
		}
		expr = number("v*100", precision) + "+'%'"
	case "currency":
		if precision < 0 {
			precision = 2
		}
		symbol := f.Currency
		if symbol == "" {
			symbol = "$"
		}
		encoded, _ := json.Marshal(symbol)
		expr = string(encoded) + "+" + number("v", precision)
	case "si":
		if precision < 0 {
			precision = 1
		}
		label.Formatter = opts.FuncOpts("function(v){var units=[[1e12,'T'],[1e9,'G'],[1e6,'M'],[1e3,'k']];for(var i=0;i<units.length;i++){if(Math.abs(v)>=units[i][0]){return " + number("v/units[i][0]", precision) + "+units[i][1];}}return " + number("v", precision) + ";}")
		return label, nil
	}
	label.Formatter = opts.FuncOpts("function(v){return " + expr + ";}")
	return label, nil
}
