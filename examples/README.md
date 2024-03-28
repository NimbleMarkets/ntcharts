# Examples

## Quickstart

This [tutorial](quickstart/README.md) creates a simple [Time Series Chart](#time-series) with two data sets utilizing the Bubble Tea framework, Lip Gloss for styling and BubbleZone for mouse support.

<p>
<a href="quickstart/main.go" alt="quickstart example">(source)</a><br>
<a href="quickstart/README.md" alt="quickstart README"><img src="quickstart/demo.gif" alt="quickstart gif" width="400"/></a>
</p>

## Canvas

A Canvas provides a 2D grid to plot arbitrary runes supporting [charmbraclet/lipgloss](https://github.com/charmbracelet/lipgloss) styles and uses [lrstanley/bubblezone](https://github.com/lrstanley/bubblezone) for mouse support.

<p>
<a href="canvas/logo/main.go" alt="logo canvas example">(source)</a><br>
<img src="canvas/logo/demo.gif" alt="logo canvas gif"/>
</p>

## Graphing

There are various graphing functions for drawing runes onto the Canvas.

### Braille

<p>
<a href="graph/braille/main.go" alt="braille graph example">(source)</a><br>
<img src="graph/braille/demo.gif" alt="braille graph gif"/>
</p>

### Circles

<p>
<a href="graph/circles/main.go" alt="circles graph example">(source)</a><br>
<img src="graph/circles/demo.gif" alt="circles graph gif"/>
</p>

### Columns

<p>
<a href="graph/columns/main.go" alt="columns graph example">(source)</a><br>
<img src="graph/columns/demo.gif" alt="columns graph gif"/>
</p>

### Lines

<p>
<a href="graph/lines/main.go" alt="lines graph example">(source)</a><br>
<img src="graph/lines/demo.gif" alt="lines graph gif"/>
</p>

### Rows

<p>
<a href="graph/rows/main.go" alt="rows graph example">(source)</a><br>
<img src="graph/rows/demo.gif" alt="rows graph gif"/>
</p>

## Bar Chart

Barcharts displays values as either horizontal rows or vertical columns.

### Rows

<p>
<a href="barchart/horizontal/main.go" alt="horizontal barchart example">(source)</a><br>
<img src="barchart/horizontal/demo.gif" alt="horizontal barchart gif"/>
</p>

### Columns

<p>
<a href="barchart/vertical/main.go" alt="vertical barchart example">(source)</a><br>
<img src="barchart/vertical/demo.gif" alt="vertical barchart gif"/>
</p>

## Line Chart

Linecharts displays (X,Y) data points onto a 2D grid in various types of charts.

### Circles

Circles can be displayed with a given point and radius.

<p>
<a href="linechart/circles/main.go" alt="circles linechart example">(source)</a><br>
<img src="linechart/circles/demo.gif" alt="circles linechart gif"/>
</p>

### Lines

Lines can be displayed between two points.

<p>
<a href="linechart/lines/main.go" alt="lines linechart example">(source)</a><br>
<img src="linechart/lines/demo.gif" alt="lines linechart gif"/>
</p>

### Scatter

Scatter charts can be created by plotting abitrary runes onto (X,Y) coordinates.

<p>
<a href="linechart/scatter/main.go" alt="scatter linechart example">(source)</a><br>
<img src="linechart/scatter/demo.gif" alt="scatter linechart gif"/>
</p>

### Streaming

Streaming charts display a continuous a line moving across the Canvas from the right side to the left side.

<p>
<a href="linechart/streaming/main.go" alt="streaming linechart example">(source)</a><br>
<img src="linechart/streaming/demo.gif" alt="streaming linechart gif"/>
</p>

### Time Series

Time series charts have values on the Y axis and time values on the X axis.

<p>
<a href="linechart/timeseries/main.go" alt="timeseries linechart example">(source)</a><br>
<img src="linechart/timeseries/demo.gif" alt="timeseries linechart gif"/>
</p>

### Wave Line

Wave line charts display a continuous a line going across the line chart.

<p>
<a href="linechart/wavelines/main.go" alt="wavelines linechart example">(source)</a><br>
<img src="linechart/wavelines/demo.gif" alt="wavelines linechart gif"/>
</p>

## Sparkline

Sparklines displays data moving across the Canvas from the right side to the left side.

<p>
<a href="sparkline/main.go" alt="sparkline example">(source)</a><br>
<img src="sparkline/demo.gif" alt="sparkline gif"/>
</p>
