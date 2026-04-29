# Canvas Text Rendering Test

## Expected Visual Output

A 20x5 canvas displaying:
- Line 1: "HELLO WORLD" in cyan color
- Line 2: "ntcharts test" in default color
- Line 3: Empty
- Line 4: "Line 4" left-aligned
- Line 5: Numbers "12345"

## Verification Criteria

1. **Text Placement**: All text should be visible and properly positioned on separate lines
2. **Color**: "HELLO WORLD" should appear in cyan/teal color
3. **Alignment**: Text should be left-aligned starting from column 0
4. **Canvas Bounds**: Content should fit within the 20x5 character grid
5. **No Artifacts**: No unexpected characters or visual glitches

## Pass Conditions

- All 4 text lines are readable
- Cyan coloring is visible on first line
- Text does not overflow or wrap incorrectly
