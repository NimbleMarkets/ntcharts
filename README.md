# bubbletea-charts

<p>
    <a href="https://github.com/NimbleMarkets/bubbletea-charts/releases"><img src="https://img.shields.io/github/release/NimbleMarkets/bubbletea-charts.svg" alt="Latest Release"></a>
    <a href="https://pkg.go.dev/github.com/NimbleMarkets/bubbletea-charts?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="GoDoc"></a>
    <a href="https://stuff.charm.sh/bubbletea/bubbletea-4k.png"><img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg"  alt="Code Of Conduct"></a>
</p>

`bubbletea-charts` is a Golang TUI Charting library for the Bubble Tea Framework.

We supply many chart types within the glory of your terminal!  See the [`examples` folder](./examples/README.md) for code samples and visuals of each type.

| Type | Description |
| :-------- | :----- |
| [Canvas](./examples/README.md#canvas) | A 2D grid to plot arbitrary runes, with [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling and [BubbleZone](https://github.com/lrstanley/bubblezone) for mousing.  It is the foundation for all the following charts. |
| [Bar Chart](./examples/README.md#bar-chart) | Displays values as either horizontal rows or vertical columns. |
| [Line Chart](./examples/README.md#lines) | Displays (X,Y) data points onto a 2D grid in various types of charts. |
| [Scatter Chart](./examples/README.md#scatter) | Plots abitrary runes onto (X,Y) coordinates. |
| [Streamline Chart](./examples/README.md#streaming) | Displays a continuous a line moving across the Canvas from the right side to the left side. |
| [Time Series Chart](./examples/README.md#time-series) | Displays lines with values on the Y axis and time values on the X axis. |
| [Waveline Chart](./examples/README.md#wave-line) | A line chart that connects points in a wave pattern. |
| [Sparkline](./examples/README.md#sparkline) | A small, simple visual of data chart for quick understanding. |

## Open Collaboration

We welcome contributions and feedback.  Please adhere to our [Code of Conduct](./CODE_OF_CONDUCT.md) when engaging our community.

 * [GitHub Issues](https://github.com/NimbleMarkets/bubbletea-charts/issues)
 * [GitHub Pull Requests](https://github.com/NimbleMarkets/bubbletea-charts/pulls)

---

## Quickstart

This tutorial creates a simple [Time Series Chart](https://github.com/NimbleMarkets/bubbletea-charts/blob/tony-branch/examples/linechart/timeseries/main.go) below that uses keyboard and mouse for zooming in and out, and moving the chart right and left.
The source code can be found at [examples/quickstart/main.go](./examples/quickstart/main.go). 

<img src="examples/quickstart/demo.gif" alt="quickstart gif" width='300'/>

First, define the package and import some libraries used for this tutorial. BubbleTea-Charts use the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling and [BubbleZone](https://github.com/lrstanley/bubblezone) for mouse support.

```go
package main

import (
    "fmt"
    "os"
    "time"

    tslc "github.com/NimbleMarkets/bubbletea-charts/linechart/timeserieslinechart"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    zone "github.com/lrstanley/bubblezone"
)
```

Create a new time series chart. The default time series chart increments the X axis by each day and automatically scales the maximum Y value.  By default, the chart zooms in and out along the X axis and moves the viewing window by day increments.

```go
func main() {
    // create new time series chart
    width := 30
    height := 12
    chart := tslc.New(width, height)

    // additional chart code goes here
}
```

Add data to default data set with `chart.Push()`.  The Lip Gloss style used when displaying the data set can be set with `chart.SetStyle()`.

```go
// add default data set
dataSet := []float64{0, 2, 4, 6, 8, 10, 8, 6, 4, 2, 0}
for i, v := range dataSet {
    date := time.Now().Add(time.Hour * time.Duration(24*i))
    chart.Push(tslc.TimePoint{date, v})
}

// set default data set line color to red
chart.SetStyle(
    lipgloss.NewStyle().
        Foreground(lipgloss.Color("9")), // red
)
```

Additional data sets can be added by name with `PushDataSet()`. The Lip Gloss style used for additional data sets can set with `chart.SetDataSetStyle()`.

```go
// add additional data set by name
dataSet2 := []float64{10, 8, 6, 4, 2, 0, 2, 4, 6, 8, 10}
for i, v := range dataSet2 {
    date := time.Now().Add(time.Hour * time.Duration(24*i))
    chart.PushDataSet("dataSet2", tslc.TimePoint{date, v})
}

// set additional data set line color to green
chart.SetDataSetStyle("dataSet2",
    lipgloss.NewStyle().
        Foreground(lipgloss.Color("10")), // green
)
```

Enable mouse support by using the BubbleZone library.  BubbleTea-Charts components requires a `zone.Manager` to enable mouse support.

```go
// mouse support is enabled with BubbleZone
zoneManager := zone.New()
chart.SetZoneManager(zoneManager)
chart.Focus() // set focus to process keyboard and mouse messages
```

Create a Bubble Tea `Model` to contain the time series chart and BubbleZone `Manager`.

```go
type model struct {
    chart       tslc.Model
    zoneManager *zone.Manager
}

func (m model) Init() tea.Cmd {
    return nil
}
```

Forward Bubble Tea keyboard and mouse events to the time series chart on `Update()`.  This example will draw all data sets using braille runes with `DrawBrailleAll()`.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    }

    // forward Bubble Tea Msg to time series chart
    // and draw all data sets using braille runes
    m.chart, _ = m.chart.Update(msg)
    m.chart.DrawBrailleAll()
    return m, nil
}
```

In the root model, wrap the time series chart `View()` output with BubbleZone `Manager.Scan()` using the `Manager` that the time series chart is set to.  A Lip Gloss style is used to apply a purple border around the chart.

```go
func (m model) View() string {
    // call bubblezone Manager.Scan() at root model
    return m.zoneManager.Scan(
        lipgloss.NewStyle().
            BorderStyle(lipgloss.NormalBorder()).
            BorderForeground(lipgloss.Color("63")). // purple
            Render(m.chart.View()),
    )
}
```

Finally, create a new Bubble Tea program with mouse cell motion enabled for mouse support and run program.

```go
func main() {
    // [...]

    // start new Bubble Tea program with mouse support enabled
    m := model{chart, zoneManager}
    if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
        fmt.Println("Error running program:", err)
        os.Exit(1)
    }
}
```

---

## Acknowledgements

Thanks to [Charm.sh](https://charm.sh) for making the command line glamorous and sharing [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) and more.

Thanks also to [ratatui](https://docs.rs/ratatui/latest/ratatui/index.html) and [termdash](https://github.com/mum4k/termdash) for inspiration.

## License

Released under the [MIT License](https://en.wikipedia.org/wiki/MIT_License), see [LICENSE.txt](./LICENSE.txt).

Copyright (c) 2024 [Neomantra Corp](https://www.neomantra.com).   

----
Made with :heart: and :fire: by the team behind [Nimble.Markets](https://nimble.markets).
