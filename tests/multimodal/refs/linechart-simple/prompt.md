# Simple Line Chart Test

## Expected Visual Output

A 40x15 line chart displaying:
- Y-axis on the left with values from 0 to 10
- X-axis on the bottom with values from 0 to 10
- A line connecting points: (0,0), (2,4), (4,8), (6,6), (8,2), (10,5)
- Axis lines forming an L-shape at the origin

## Verification Criteria

1. **Axes**: Y-axis vertical line visible on left, X-axis horizontal line visible on bottom
2. **Labels**: Numeric labels visible on both axes
3. **Line Drawing**: Connected line segments visible between data points
4. **Data Accuracy**: Line should rise from (0,0) to peak around (4,8), drop to (8,2), then rise to (10,5)
5. **Scale**: Chart should use the full plotting area

## Pass Conditions

- Both axes are clearly visible with labels
- Line pattern shows: rise, peak, fall, small rise
- No visual artifacts or missing segments
