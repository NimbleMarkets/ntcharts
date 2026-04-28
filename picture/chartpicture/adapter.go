package chartpicture

import (
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/go-analyze/charts"
)

// BarChartOptionFromNT builds a charts.BarChartOption from an ntcharts
// barchart.Model. It pulls the bar data via the new Data() accessor and
// mirrors visual configuration (orientation, axis visibility) onto the
// returned option.
func BarChartOptionFromNT(bc *barchart.Model) charts.BarChartOption {
	if bc == nil {
		return charts.NewBarChartOptionWithSeries(nil)
	}

	bd := bc.Data()
	series := buildBarSeriesList(bd)
	labels := make([]string, len(bd))
	for i, b := range bd {
		labels[i] = b.Label
	}

	opt := charts.NewBarChartOptionWithSeries(series)
	opt.Horizontal = bc.Horizontal()
	opt.CategoryAxis = charts.CategoryAxisOption{Labels: labels}

	if !bc.ShowAxis() {
		show := false
		opt.ValueAxis = []charts.ValueAxisOption{{Show: &show}}
		opt.CategoryAxis.Show = &show
	}
	if max := bc.MaxValue(); max > 0 {
		if len(opt.ValueAxis) == 0 {
			opt.ValueAxis = []charts.ValueAxisOption{{}}
		}
		opt.ValueAxis[0].Max = charts.Ptr(max)
	}
	return opt
}

// buildBarSeriesList walks bd in order, collecting each unique value Name
// into its own charts.BarSeries with values aligned to bd's index. When a
// BarData has multiple values, each named value contributes to its own series.
func buildBarSeriesList(bd []barchart.BarData) charts.BarSeriesList {
	if len(bd) == 0 {
		return nil
	}
	var order []string
	seen := map[string]int{}
	for _, b := range bd {
		for _, v := range b.Values {
			if _, ok := seen[v.Name]; !ok {
				seen[v.Name] = len(order)
				order = append(order, v.Name)
			}
		}
	}
	series := make(charts.BarSeriesList, len(order))
	for i, name := range order {
		series[i].Name = name
		series[i].Values = make([]float64, len(bd))
	}
	for i, b := range bd {
		for _, v := range b.Values {
			si := seen[v.Name]
			series[si].Values[i] = v.Value
		}
	}
	return series
}

// LineChartOptionFromNT builds a charts.LineChartOption from an ntcharts
// linechart.Model plus parallel series data and names. linechart.Model is a
// drawing canvas — it does not store the underlying series — so callers must
// supply data alongside.
func LineChartOptionFromNT(lc *linechart.Model, series [][]float64, names []string) charts.LineChartOption {
	sl := make(charts.LineSeriesList, len(series))
	for i, vals := range series {
		sl[i].Values = append([]float64(nil), vals...)
		if i < len(names) {
			sl[i].Name = names[i]
		}
	}
	opt := charts.NewLineChartOptionWithSeries(sl)
	if lc != nil {
		opt.YAxis = []charts.YAxisOption{{
			Min: charts.Ptr(lc.MinY()),
			Max: charts.Ptr(lc.MaxY()),
		}}
	}
	return opt
}
