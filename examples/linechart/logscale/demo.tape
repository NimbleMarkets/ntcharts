Output demo.gif

Require echo

Set Shell "bash"
Set Padding 0
Set Framerate 30
Set FontSize 20
Set Width 850
Set Height 540

Hide
Type "./bin/ntcharts-linechart-logscale"
Enter
Show

# linear axis: the growth is squashed into the last few columns
Sleep 1.5s

# log axis: labels on powers of ten, steady growth
Type "l"
Sleep 2s

# zoom the Y axis: each step multiplies the range
Type@400ms "+++"
Sleep 1s
Type@400ms "---"
Sleep 800ms

# and back to linear
Type "l"
Sleep 1.5s

Hide
Type "q"
