package heatpicture

import "image/color"

// colorAt returns the color at normalized position t (clamped to [0,1]) on
// the given pre-converted RGBA scale, linearly interpolating between
// adjacent stops in straight RGBA space. An empty scale yields fully
// transparent — the renderer treats that as "no scale set yet" and
// produces a transparent frame rather than crashing.
//
// Takes []color.RGBA (not []color.Color) so the per-pixel hot path avoids
// interface boxing and the repeated 16-bit-to-8-bit shift on each call;
// SetColorScale precomputes the conversion once per scale change.
func colorAt(scale []color.RGBA, t float64) color.RGBA {
	if len(scale) == 0 {
		return color.RGBA{0, 0, 0, 0}
	}
	if len(scale) == 1 {
		return scale[0]
	}
	if t <= 0 {
		return scale[0]
	}
	if t >= 1 {
		return scale[len(scale)-1]
	}

	pos := t * float64(len(scale)-1)
	lo := int(pos)
	if lo >= len(scale)-1 {
		return scale[len(scale)-1]
	}
	frac := pos - float64(lo)

	return lerpRGBA(scale[lo], scale[lo+1], frac)
}

// lerpRGBA linearly interpolates between two RGBA colors in straight RGBA.
func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: uint8(float64(a.A)*(1-t) + float64(b.A)*t),
	}
}

// toRGBA precomputes a []color.RGBA copy of a user-provided []color.Color
// so the per-pixel hot path can index into a value-typed slice without
// re-running .RGBA() and the 16→8-bit shift on every pixel.
func toRGBA(scale []color.Color) []color.RGBA {
	out := make([]color.RGBA, len(scale))
	for i, c := range scale {
		r, g, b, a := c.RGBA()
		out[i] = color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8),
		}
	}
	return out
}
