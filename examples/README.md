# Examples

## Canvas

A Canvas provides a 2D grid to plot arbitrary runes supporting [charmbraclet/lipgloss](https://github.com/charmbracelet/lipgloss) styles and uses [lrstanley/bubblezone](https://github.com/lrstanley/bubblezone) for mouse support.

[(source)](./canvas/logo/main.go)
<img src="canvas/logo/demo.gif" alt="canvas logo gif"/>

## Graphing

There are various graphing functions for drawing runes onto the Canvas.

### Braille

[(source)](./graph/braille/main.go)
<img src="graph/braille/demo.gif" alt="graph braille gif"/>

### Circles

[(source)](./graph/circles/main.go)
<img src="graph/circles/demo.gif" alt="graph circles gif"/>

### Columns

[(source)](./graph/columns/main.go)
<img src="graph/columns/demo.gif" alt="graph columns gif"/>

### Lines

[(source)](./graph/lines/main.go)
<img src="graph/lines/demo.gif" alt="graph lines gif"/>

### Rows

[(source)](./graph/rows/main.go)
<img src="graph/rows/demo.gif" alt="graph rows gif"/>

## Bar Chart

Barcharts displays values as either horizontal rows or vertical columns.

### Rows

[(source)](./barchart/horizontal/main.go)
<img src="barchart/horizontal/demo.gif" alt="barchart horizontal gif"/>

### Columns

[(source)](./barchart/vertical/main.go)
<img src="barchart/vertical/demo.gif" alt="barchart vertical gif"/>

## Line Chart

Linecharts displays (X,Y) data points onto a 2D grid in various types of charts.

### Circles

Circles can be displayed with a given point and radius.

[(source)](./linechart/circles/main.go)
<img src="linechart/circles/demo.gif" alt="linechart circles gif"/>

### Lines

Lines can be displayed between two points.

[(source)](./linechart/lines/main.go)
<img src="linechart/lines/demo.gif" alt="linechart lines gif"/>

### Scatter

Scatter charts can be created by plotting abitrary runes onto (X,Y) coordinates.

[(source)](./linechart/scatter/main.go)
<img src="linechart/scatter/demo.gif" alt="linechart scatter gif"/>

### Streaming

Streaming charts display a continuous a line moving across the Canvas from the right side to the left side.

[(source)](./linechart/streaming/main.go)
<img src="linechart/streaming/demo.gif" alt="linechart streaming gif"/>

### Time Series

Time series charts have values on the Y axis and time values on the X axis.

[(source)](./linechart/timeseries/main.go)
<img src="linechart/timeseries/demo.gif" alt="linechart timeseries gif"/>

### Wave Line

Wave line charts display a continuous a line going across the line chart.

[(source)](./linechart/wavelines/main.go)
<img src="linechart/wavelines/demo.gif" alt="linechart wavelines gif"/>

## Sparkline

Sparklines displays data moving across the Canvas from the right side to the left side.

[(source)](./sparkline/main.go)
<img src="sparkline/demo.gif" alt="sparkline gif"/>
