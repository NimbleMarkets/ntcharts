
# Quickstart Tutorial

This tutorial creates a simple `timeserieslinechart` that uses keyboard and mouse for zooming in and out, and moving the chart right and left.
The source code can be found at [main.go](./main.go). 

<img src="./demo.gif" alt="quickstart gif" width='300'/>

First, define the package and import some libraries used for this tutorial. ntcharts use the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling and [BubbleZone](https://github.com/lrstanley/bubblezone) for mouse support.

```go
package main

import (
    "fmt"
    "os"
    "time"

    tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"

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

Enable mouse support by using the BubbleZone library.  ntcharts components requires a `zone.Manager` to enable mouse support.

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
