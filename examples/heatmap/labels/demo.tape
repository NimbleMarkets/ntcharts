Output demo.gif

Require echo

Set Shell "bash"
Set Padding 0
Set Framerate 30
Set FontSize 20
Set Width 1000
Set Height 520

Hide
Type "./bin/ntcharts-heatmap-labels"
Enter
Show

# a labelled grid: days down the side, time of day along the bottom
Sleep 2s

# no labels: the cells fill the whole chart
Type "l"
Sleep 1.5s
Type "l"
Sleep 1s

# shrink the chart: labels drop out rather than collide
Type@500ms "----"
Sleep 1.2s

# and grow it again
Type@500ms "++++++"
Sleep 1.5s

Hide
Type "q"
