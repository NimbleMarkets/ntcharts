Output demo.gif

Require echo

Set Shell "bash"
Set Padding 0
Set Framerate 30
Set FontSize 20
Set Width 1600
Set Height 600

Hide
Type "./bin/btcharts-ohlc -filepath cmd/btcharts-ohlc/example.csv -high -low -vol -braille"
Enter
Show

Right@150ms 20

Sleep 150ms

Hide
Type "q"
