# Basic Sparkline Test

## Expected Visual Output

A compact sparkline visualization (20x5) showing:
- A series of block characters representing values: 1, 3, 5, 7, 9, 7, 5, 3, 1
- Rising pattern from left to center, falling pattern from center to right
- Peak in the middle of the sparkline

## Verification Criteria

1. **Block Characters**: Sparkline uses block characters (▁▂▃▄▅▆▇█) of varying heights
2. **Pattern**: Values rise from 1 to 9 then fall back to 1 (mountain shape)
3. **Symmetry**: Left and right halves should mirror each other
4. **Compact**: Fits within the specified dimensions

## Pass Conditions

- Block characters of varying heights visible
- Clear mountain/peak pattern with highest point in center
- Pattern is visually symmetric
