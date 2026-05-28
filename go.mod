module github.com/NimbleMarkets/ntcharts/v2

go 1.25.0

// Awaiting upstream merges
replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20260506185856-6506c47fa2f3

require (
	charm.land/bubbles/v2 v2.1.0
	charm.land/bubbletea/v2 v2.0.6
	charm.land/lipgloss/v2 v2.0.3
	github.com/NimbleMarkets/go-booba v0.6.1-0.20260511134559-58814d532cc1
	github.com/NimbleMarkets/pixterm v0.0.0-20260501211346-dc18ac6c1a0f
	github.com/aquilax/go-perlin v1.1.0
	github.com/charmbracelet/ultraviolet v0.0.0-20260525132238-948f4557a654
	github.com/go-analyze/charts v0.5.27
	github.com/lrstanley/bubblezone/v2 v2.0.0
	github.com/spf13/pflag v1.0.10
	golang.org/x/image v0.41.0
	gopkg.in/yaml.v3 v3.0.1
)

tool github.com/NimbleMarkets/go-booba/cmd/booba-assets

require (
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/x/ansi v0.11.7
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-analyze/bulk v0.1.4 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)
