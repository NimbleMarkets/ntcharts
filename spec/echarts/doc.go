// ntcharts - Copyright (c) 2026 Neomantra Corp.

// Package echarts renders a spec.Spec as an Apache ECharts web chart using
// go-echarts/v2.
//
// It is the web surface of the ntcharts spec package: the same Spec that
// spec.Build renders to a terminal model can be handed to ToECharts to obtain
// a go-echarts chart with a Render(io.Writer) method.
//
// This package lives in its own Go module,
// github.com/NimbleMarkets/ntcharts/spec/echarts/v2, so that the go-echarts
// dependency is only pulled in by programs that render to the web. Terminal
// consumers of github.com/NimbleMarkets/ntcharts/v2/spec are unaffected.
//
// Only spec.ChartTypeBar and spec.ChartTypeTimeSeries are implemented today;
// every other chart type returns a "not yet implemented" error. See the
// surface fidelity matrix in spec/README.md for which Spec options each
// surface honours.
package echarts
