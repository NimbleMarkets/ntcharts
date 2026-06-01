module github.com/NimbleMarkets/ntcharts/v2/cmd

go 1.25.0

require (
	charm.land/bubbles/v2 v2.1.0
	charm.land/bubbletea/v2 v2.0.7
	charm.land/lipgloss/v2 v2.0.3
	github.com/NimbleMarkets/ntcharts/v2 v2.2.0
	github.com/lrstanley/bubblezone/v2 v2.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/NimbleMarkets/go-booba v0.6.1-0.20260511134559-58814d532cc1 // indirect
	github.com/NimbleMarkets/pixterm v0.0.0-20260501211346-dc18ac6c1a0f // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260601155805-6cf7526a1b3f // indirect
	github.com/charmbracelet/x/ansi v0.11.7 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/image v0.41.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

tool github.com/NimbleMarkets/go-booba/cmd/booba-assets

replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20260506185856-6506c47fa2f3

replace github.com/NimbleMarkets/ntcharts/v2 => ..
