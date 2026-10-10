Output demo.gif

Require echo

Set Shell "bash"
Set Padding 0
Set Framerate 30
Set FontSize 20
Set Width 1000
Set Height 540

Hide
Type "./bin/ntcharts-linechart-dualaxis"
Enter
Show

# load on the left axis, temperature on the right
Sleep 2s

# zoom the Y axes: the right axis follows the left
Type@400ms "+++"
Sleep 1s
Type@400ms "---"
Sleep 800ms

# zoom into the time axis, then move back along it
PageUp@400ms 3
Sleep 800ms
Type@300ms "hhhh"
Sleep 1s

Hide
Type "q"
