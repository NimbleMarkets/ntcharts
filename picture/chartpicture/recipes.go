package chartpicture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"

	"github.com/go-analyze/charts"
)

// renderRecipe rasterizes a chart at the given pixel size and theme into PNG
// bytes. Each setter on Model converts its input into one of these closures.
type renderRecipe func(widthPx, heightPx int, theme string) ([]byte, error)

func newPainter(w, h int, theme string) *charts.Painter {
	po := charts.PainterOptions{
		OutputFormat: "png",
		Width:        w,
		Height:       h,
	}
	if theme != "" {
		po.Theme = charts.GetTheme(theme)
	}
	return charts.NewPainter(po)
}

func newLineRecipe(opt charts.LineChartOption) renderRecipe {
	return func(w, h int, theme string) ([]byte, error) {
		p := newPainter(w, h, theme)
		if err := p.LineChart(opt); err != nil {
			return nil, fmt.Errorf("chartpicture: render line chart: %w", err)
		}
		buf, err := p.Bytes()
		if err != nil {
			return nil, fmt.Errorf("chartpicture: encode PNG: %w", err)
		}
		return buf, nil
	}
}

func newBarRecipe(opt charts.BarChartOption) renderRecipe {
	return func(w, h int, theme string) ([]byte, error) {
		p := newPainter(w, h, theme)
		if err := p.BarChart(opt); err != nil {
			return nil, fmt.Errorf("chartpicture: render bar chart: %w", err)
		}
		buf, err := p.Bytes()
		if err != nil {
			return nil, fmt.Errorf("chartpicture: encode PNG: %w", err)
		}
		return buf, nil
	}
}

// newEChartsJSONRecipe parses jsonStr as an ECharts option, converts it via
// EChartsOption.ToOption(), and renders via charts.Render so we can override
// width/height/theme regardless of what the JSON specified.
func newEChartsJSONRecipe(jsonStr string) renderRecipe {
	return func(w, h int, theme string) ([]byte, error) {
		var eo charts.EChartsOption
		if err := json.Unmarshal([]byte(jsonStr), &eo); err != nil {
			return nil, fmt.Errorf("chartpicture: parse ECharts JSON: %w", err)
		}
		opt := eo.ToOption()
		funcs := []charts.OptionFunc{
			charts.DimensionsOptionFunc(w, h),
			charts.PNGOutputOptionFunc(),
		}
		if theme != "" {
			funcs = append(funcs, charts.ThemeNameOptionFunc(theme))
		}
		p, err := charts.Render(opt, funcs...)
		if err != nil {
			return nil, fmt.Errorf("chartpicture: render ECharts: %w", err)
		}
		buf, err := p.Bytes()
		if err != nil {
			return nil, fmt.Errorf("chartpicture: encode PNG: %w", err)
		}
		return buf, nil
	}
}

func newPainterFuncRecipe(f func(*charts.Painter) error) renderRecipe {
	return func(w, h int, theme string) ([]byte, error) {
		p := newPainter(w, h, theme)
		if err := f(p); err != nil {
			return nil, fmt.Errorf("chartpicture: painter func: %w", err)
		}
		buf, err := p.Bytes()
		if err != nil {
			return nil, fmt.Errorf("chartpicture: encode PNG: %w", err)
		}
		return buf, nil
	}
}

func decodePNGToImage(buf []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(buf))
}
