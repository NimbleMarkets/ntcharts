# Examples

When you run `task`, all of the examples are built into the the `bin` directory of `ntcharts`.  So run any example below like so: `./bin/ntcharts-quickstart`.

## Quickstart

This [tutorial](quickstart/README.md) creates a simple [Time Series Chart](#time-series) with two data sets utilizing the Bubble Tea framework, Lip Gloss for styling and BubbleZone for mouse support.

`ntcharts-quickstart` [(source)](./quickstart/main.go) <br>
<img src="quickstart/demo.gif" alt="quickstart gif" width="400"/>

## Canvas

A Canvas provides a 2D grid to plot arbitrary runes supporting [Lip Gloss](https://github.com/charmbracelet/lipgloss) styles and uses [BubbleZone](https://github.com/lrstanley/bubblezone) for mouse support.

`ntcharts-canvas-logo` [(source)](./canvas/logo/main.go)<br>
<img src="canvas/logo/demo.gif" alt="logo canvas gif"/>

## Graphing

There are various graphing functions for drawing runes onto the Canvas.

### Braille

`ntcharts-graph-braille` [(source)](./graph/braille/main.go)<br>
<img src="graph/braille/demo.gif" alt="braille graph gif"/>

### Candlesticks

`ntcharts-graph-candlesticks` [(source)](./graph/candlesticks/main.go)<br>
<img src="graph/candlesticks/demo.gif" alt="candlesticks graph gif"/>

### Circles

`ntcharts-graph-circles` [(source)](./graph/circles/main.go)<br>
<img src="graph/circles/demo.gif" alt="circles graph gif"/>

### Columns

`ntcharts-graph-columns` [(source)](./graph/columns/main.go)<br>
<img src="graph/columns/demo.gif" alt="columns graph gif"/>

### Lines

`ntcharts-graph-lines` [(source)](./graph/lines/main.go)<br>
<img src="graph/lines/demo.gif" alt="lines graph gif"/>

### Rows

`ntcharts-graph-rows` [(source)](./graph/rows/main.go)<br>
<img src="graph/rows/demo.gif" alt="rows graph gif"/>

## Bar Chart

Barcharts displays values as either horizontal rows or vertical columns.

### Rows

`ntcharts-barchart-horizontal` [(source)](./barchart/horizontal/main.go)<br>
<img src="barchart/horizontal/demo.gif" alt="horizontal barchart gif"/>

### Columns

`ntcharts-barchart-vertical` [(source)](./barchart/vertical/main.go)<br>
<img src="barchart/vertical/demo.gif" alt="vertical barchart gif"/>

## Line Chart

Linecharts displays (X,Y) data points onto a 2D grid in various types of charts.

### Circles

Circles can be displayed with a given point and radius.

`ntcharts-linechart-circles` [(source)](./linechart/circles/main.go)<br>
<img src="linechart/circles/demo.gif" alt="circles linechart gif"/>

### Lines

Lines can be displayed between two points.

`ntcharts-linechart-lines` [(source)](./linechart/lines/main.go)<br>
<img src="linechart/lines/demo.gif" alt="lines linechart gif"/>

### Scatter

Scatter charts can be created by plotting abitrary runes onto (X,Y) coordinates.

`ntcharts-linechart-scatter` [(source)](./linechart/scatter/main.go)<br>
<img src="linechart/scatter/demo.gif" alt="scatter linechart gif"/>

### Streaming

Streaming charts display a continuous a line moving across the Canvas from the right side to the left side.

`ntcharts-linechart-streaming` [(source)](./linechart/streaming/main.go)<br>
<img src="linechart/streaming/demo.gif" alt="streaming linechart gif"/>

### Time Series

Time series charts have values on the Y axis and time values on the X axis.

`ntcharts-linechart-timeseries` [(source)](./linechart/timeseries/main.go)<br>
<img src="linechart/timeseries/demo.gif" alt="timeseries linechart gif"/>

### Wave Line

Wave line charts display a continuous a line going across the line chart.

`ntcharts-linechart-wavelines` [(source)](./linechart/wavelines/main.go)<br>
<img src="linechart/wavelines/demo.gif" alt="wavelines linechart gif"/>

## Sparkline

Sparklines displays data moving across the Canvas from the right side to the left side.

`ntcharts-sparkline` [(source)](./sparkline/main.go)<br>
<img src="sparkline/demo.gif" alt="sparkline gif"/>

## Heatmap

Heatmaps display (x,y) data over a grid with color mapping. 

`ntcharts-heatmap-simple` [Simple Heatmap (source)](./heatmap/simple/main.go)<br>
<img src="usage/heatmap.png" alt="simple heatmap image" width='100'/>

`ntcharts-heatmap-aoc2024` [AoC 2024.4 Heatmap (source)](./heatmap/aoc2024/main.go)<br>
<img src="heatmap/aoc2024/aoc2024_4.jpg" alt="heatmap aoc2024 image" width="300"/>

`ntcharts-heatmap-functor` [Functors TUI(source)](./heatmap/functors/main.go)<br>
<img src="heatmap/functor/functor1.png" alt="heatmap functors still" width="200"/>
<img src="heatmap/functor/functor2.png" alt="heatmap functors still" width="200"/>

`ntcharts-heatmap-perlin` [Perlin Noise TUI (source)](./heatmap/perlin/main.go)<br>
<img src="heatmap/perlin/perlin.png" alt="heatmap perlin still" width="300"/>