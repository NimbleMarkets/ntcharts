// Package chartpicture is a go-analyze/charts source for picture.Model.
//
// Each setter (SetLineChartOption, SetBarChartOption, SetEChartsJSON,
// SetPainterFunc) converts its input into a single internal "render recipe"
// closure. SetSize and SetTheme re-invoke the stored recipe. Rendering runs
// off the main loop via tea.Cmd; the resulting PNG is decoded to image.Image
// and handed to the embedded picture.Model.
//
// Mirrors the API shape of picture/pictureurl: typed setters returning
// tea.Cmd, async render, seq-based stale-frame protection, value
// constructors with pointer-receiver methods.
package chartpicture
