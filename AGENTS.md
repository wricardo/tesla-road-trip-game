# Agent Instructions for Tesla Road Trip Game

## CRITICAL: Grid Character Recognition

### The #1 Problem: Misreading 'R' as 'B' or 'W'

Agents frequently fail to recognize road characters ('R') when they appear between obstacles. This document provides explicit instructions to prevent this critical error.

## Character Reference

```
R = Road (PASSABLE) - You CAN move here
H = Home (PASSABLE) - You CAN move here + full battery charge
P = Park (PASSABLE) - You CAN move here + collect park
S = Supercharger (PASSABLE) - You CAN move here + full battery charge
W = Water (IMPASSABLE) - You CANNOT move here
B = Building (IMPASSABLE) - You CANNOT move here
```

## Common Misreading Examples

### Example 1: Hidden Road Between Buildings
```
What you might see: BBBBBWWWWWBBBBB
What it actually is: BBBBRWWWWWBBBBB
                          ^
                          This is an R (road), not B!
```

### Example 2: Road at Start of Water
```
What you might see: WWWWWWBBBBB
What it actually is: RWWWWWBBBBB
                      ^
                      This is an R (road), not W!
```

### Example 3: Single Road in Building Cluster
```
What you might see: BBBBBBBBBB
What it actually is: BBBBRBBBBBB
                          ^
                          This is an R (road)!
```

## Mandatory Grid Analysis Protocol

When analyzing any grid row, you MUST:

1. **Initial Display**
   ```
   Row 8: BBBBRWWWWWBBBBB
   ```

2. **Character-by-Character Breakdown**
   ```
   Position 0: B (building - impassable)
   Position 1: B (building - impassable)
   Position 2: B (building - impassable)
   Position 3: B (building - impassable)
   Position 4: R (road - PASSABLE!) ← PAY ATTENTION!
   Position 5: W (water - impassable)
   Position 6: W (water - impassable)
   ...continue for all positions
   ```

3. **Verification**
   - If you think a row is completely blocked, STOP
   - Re-examine each character individually
   - Look specifically for 'R' characters
   - Test movement to suspected road positions

## Decision Tree for Navigation

```
Is the row completely blocked?
├── NO → Proceed with navigation
└── YES → WAIT! Are you sure?
          ├── Re-parse character by character
          ├── Look for 'R' between B's and W's
          ├── Found an R? → It's NOT blocked!
          └── Still no R after careful check? → Row is blocked
```

## Testing Protocol

When you encounter a "blocked" row:

1. **Visual Re-check**
   - Copy the exact row string
   - Parse each character with its index
   - Mark any R characters found

2. **Use the describe_cell Tool** (NEW!)
   - Call `describe_cell(session_id, x, y)` for any suspicious cell
   - This tool will tell you EXACTLY what character is at that position
   - Eliminates all ambiguity between R, B, and W characters
   - Example: `describe_cell("abc1", 4, 8)` to check position (4, 8)

3. **Movement Verification**
   - Try moving to positions where R might be
   - Even if it looks like B or W, test it
   - Movement success confirms it's an R

4. **Documentation**
   ```
   Row 8 analysis:
   Original: BBBBRWWWWWBBBBB
   Parsed:   B B B B R W W W W W B B B B B
   Index:    0 1 2 3 4 5 6 7 8 9 10 11 12 13 14
   Roads at: Position 4
   ```

## Success Metrics

Your character recognition is successful when:
- ✅ You identify ALL road (R) positions in the grid
- ✅ You never confuse R with B or W
- ✅ You systematically verify "blocked" rows
- ✅ You can navigate through apparent obstacles via hidden roads

## NEW TOOL: describe_cell

A powerful new MCP tool has been added specifically to help with character recognition:

### Usage
```
describe_cell(session_id: str, x: int, y: int)
```

### Returns
- Exact character at the position (R, B, W, H, S, P, T, ✓)
- Cell type (Road, Building, Water, Home, Supercharger, Park)
- Whether the cell is passable (true/false)
- Helpful reminders about common confusions

### Example Response
```
Cell at position (4, 8):
━━━━━━━━━━━━━━━━━━━━━━━━
Character: R
Type: Road
Passable: true
Description: Empty road - safe to travel

IMPORTANT: The character 'R' is what appears in the grid display.
⚠️ REMINDER: 'R' (road) is often confused with 'B' (building). This is a ROAD and is PASSABLE!
```

### When to Use
- **ALWAYS** when you're unsure if a character is R, B, or W
- When a row appears "completely blocked"
- Before declaring an area impassable
- To verify your grid parsing is correct
- When movement fails unexpectedly

## Remember

**The difference between success and failure often comes down to recognizing a single 'R' character hidden among B's and W's. When in doubt, use describe_cell to get the definitive answer!**
