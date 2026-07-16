// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// gradientSteps is the resolution of an interpolated color scale.
const gradientSteps = 32

// gradientScale linearly interpolates ordered "#rrggbb" stops into a
// gradientSteps-long color scale for heatmap rendering. nil/empty stops
// return (nil, nil) so callers fall back to the chart default.
func gradientScale(stops []string) ([]color.Color, error) {
	if len(stops) == 0 {
		return nil, nil
	}
	rgbs := make([][3]float64, len(stops))
	for i, s := range stops {
		r, g, b, err := parseHexColor(s)
		if err != nil {
			return nil, err
		}
		rgbs[i] = [3]float64{r, g, b}
	}
	if len(rgbs) == 1 {
		rgbs = append(rgbs, rgbs[0])
	}
	out := make([]color.Color, gradientSteps)
	segs := len(rgbs) - 1
	for i := range out {
		t := float64(i) / float64(gradientSteps-1) * float64(segs)
		seg := int(t)
		if seg >= segs {
			seg = segs - 1
		}
		frac := t - float64(seg)
		a, b := rgbs[seg], rgbs[seg+1]
		out[i] = color.RGBA{
			R: uint8(a[0] + (b[0]-a[0])*frac),
			G: uint8(a[1] + (b[1]-a[1])*frac),
			B: uint8(a[2] + (b[2]-a[2])*frac),
			A: 0xff,
		}
	}
	return out, nil
}

func parseHexColor(s string) (r, g, b float64, err error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) != 6 {
		return 0, 0, 0, fmt.Errorf("spec: gradient stop %q is not #rrggbb", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("spec: gradient stop %q is not #rrggbb", s)
	}
	return float64((v >> 16) & 0xff), float64((v >> 8) & 0xff), float64(v & 0xff), nil
}
