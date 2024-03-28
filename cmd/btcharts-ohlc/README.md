# btcharts-ohlc

`btcharts-ohcl` displays OHLC data as a line chart from an input CSV file.  The command can display the braille lines or continuous line and choose which lines to display.

The input CSV file is required to have column headers `Date,Open,High,Low,Close,Adj Close,Volume`.  The `Date` value format is required to be in the format `YYYY-MM-DD` and in chronological order.

[(source)](./main.go/main.go)
<img src="demo.gif" alt="btcharts-ohcl gif"/>

```
./bin/btcharts-ohlc -filepath cmd/btcharts-ohlc/example.csv -high -low -vol -braille
```