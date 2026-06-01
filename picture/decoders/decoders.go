// Package decoders is a batteries-included, import-for-side-effect helper that
// registers a broad set of image decoders with the standard image package.
//
// The core picture package and picture/pictureurl register no decoders
// themselves — picture.Model renders an image.Image and pictureurl is left
// decoder-agnostic — so that importers only pay for the formats they actually
// use. Blank-import this package to opt into the full set in one line:
//
//	import _ "github.com/NimbleMarkets/ntcharts/v2/picture/decoders"
//
// After that, image.Decode handles PNG, JPEG, GIF, WebP, BMP, and TIFF, whether
// you feed picture.Model your own decoded image or let pictureurl fetch URLs.
// Developers who want a narrower set simply skip this package and register the
// formats they need (e.g. import _ "image/png").
//
// AVIF and HEIC are intentionally excluded: they require CGO or heavy external
// dependencies, which is at odds with this library's pure-Go (and WASM) pipeline.
package decoders

import (
	_ "image/gif"  // GIF decoder registration
	_ "image/jpeg" // JPEG decoder registration
	_ "image/png"  // PNG decoder registration

	_ "golang.org/x/image/bmp"  // BMP decoder registration
	_ "golang.org/x/image/tiff" // TIFF decoder registration
	_ "golang.org/x/image/webp" // WebP decoder registration
)
