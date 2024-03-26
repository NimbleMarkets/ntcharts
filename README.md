# bubbletea-charts

`bubbletea-charts` is a Golang TUI Charting library for the BubbleTea Framework.

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


## Acknowledgements

Thanks to [Charm.sh](https://charm.sh) for making the command line glamorous and sharing [BubbleTea](https://github.com/charmbracelet/bubbletea) and [LipGloss](https://github.com/charmbracelet/lipgloss) and more.

Thanks also to [ratatui](https://docs.rs/ratatui/latest/ratatui/index.html) and [termdash](https://github.com/mum4k/termdash) for inspiration.

## License

Released under the [MIT License](https://en.wikipedia.org/wiki/MIT_License), see [LICENSE.txt](./LICENSE.txt).

Copyright (c) 2024 [Neomantra Corp](https://www.neomantra.com).   

----
Made with :heart: and :fire: by the team behind [Nimble.Markets](https://nimble.markets).