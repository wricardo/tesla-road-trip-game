# Advanced Tesla Game Strategies

Detailed techniques for complex obstacle navigation, optimization patterns, and mastery-level gameplay.

## Table of Contents
- Character Recognition Deep Dive
- Advanced Pathfinding Algorithms
- Battery Optimization Techniques
- Obstacle Pattern Recognition
- Multi-Section Route Planning
- Recovery from Failed Routes

## Character Recognition Deep Dive

### The R vs B vs W Problem

In monospace terminal output, these characters share similar vertical strokes that can blur together:

```
B = Two vertical strokes with rounded bumps on the right
R = Similar to B but with a diagonal leg extending from the middle
W = Two V shapes side by side creating vertical strokes
```

### Systematic Verification Protocol

**When analyzing a grid row that appears blocked:**

1. **Copy the row string to isolate it**: `BBBBRWWWWWBBBBB`
2. **Add position markers**:
   ```
   0123456789...
   BBBBRWWWWWBBBBB
   ```
3. **Parse each position individually**:
   - Position 0: B
   - Position 1: B
   - Position 2: B
   - Position 3: B
   - Position 4: R ← This is the passage!
   - Position 5: W
   - ...

4. **Use describe_cell for verification**:
   ```
   describe_cell(session_id, x=4, y=7)
   ```

### High-Risk Misread Patterns

**Pattern 1: Single road between obstacles**
- Visual: `BBBBBBBBB` (looks completely blocked)
- Reality: `BBBBRBBBBB` (road at position 4)
- Solution: When movement is expected but appears blocked, re-parse position by position

**Pattern 2: Road-water boundary**
- Visual: `WWWWWWWWW` (looks like continuous water)
- Reality: `RWWWWWWWW` (road at position 0)
- Solution: Check boundary positions (0 and max) specifically

**Pattern 3: Clustered obstacles with gaps**
- Visual: `BBBBBBBBBBBB` (dense building cluster)
- Reality: `BBBBRBBBRBBBB` (two roads at positions 4 and 8)
- Solution: Use describe_cell on every 3rd-4th position when paths seem impossible

## Advanced Pathfinding Algorithms

### Corridor Mapping Technique

**Goal**: Identify all safe navigation corridors before planning any routes.

**Process**:
1. Scan each row left-to-right, marking continuous R/H/S/P sequences
2. Scan each column top-to-bottom, marking continuous passable sequences
3. Identify "golden corridors": rows/columns with 80%+ passable cells
4. Map corridor intersections as "hubs" for route changes

**Example Grid Analysis**:
```
Row 1: 70% passable (partial corridor)
Row 4: 90% passable (golden corridor) ← Use for horizontal travel
Row 7: 95% passable (golden corridor) ← Primary highway
Row 10: 85% passable (golden corridor)

Column 4: 60% passable (water gap at rows 6-8)
Column 7: 95% passable (golden corridor) ← Use for vertical travel
Column 9: 40% passable (heavy obstacles)
```

**Route Planning with Corridors**:
- Use Row 7 for horizontal movement
- Use Column 7 for vertical movement
- Change between corridors at intersection (7,7)

### Distance Field Calculation

**Goal**: Know the minimum battery cost to reach any charging station from any position.

**Process**:
1. Mark all charging stations (H, S) as distance 0
2. Mark all adjacent passable cells as distance 1
3. Expand outward, incrementing distance for each cell
4. Result: Every position shows battery cost to nearest charger

**Usage**:
- Never venture into cells with distance > current_battery - 3
- Prioritize objectives in low-distance zones
- Plan charging stops when entering high-distance zones

### Multi-Path Route Planning

**Goal**: Have backup routes for every objective.

**Process**:
1. Identify primary route (shortest path)
2. Identify secondary route (alternative corridor)
3. Identify tertiary route (perpendicular approach)

**Example for reaching Park at (10,4)**:
- Primary: Row 7 → Column 10 → Row 4 (15 moves)
- Secondary: Column 7 → Row 4 → along Row 4 (17 moves)
- Tertiary: Row 1 → Column 10 → Row 4 (20 moves)

**When to Switch Routes**:
- Primary blocked: Try secondary immediately
- Secondary blocked: Analyze obstacle pattern, may need tertiary
- All routes blocked: Return to charger, reassess grid analysis

## Battery Optimization Techniques

### Minimum Battery Calculation

**Formula**: `Required Battery = Distance to Objective + Distance to Nearest Charger + Safety Buffer`

**Example**:
- Current position to Park(4,10): 12 moves
- Park(4,10) to Charger(4,10): 0 moves (charger at park!)
- Safety buffer: 3
- Minimum battery needed: 15

**Decision**: With 20/20 battery, proceed confidently. With 14/20 battery, recharge first.

### Charging Station Sequencing

**Goal**: Minimize total charging stops while maintaining safety.

**Greedy Approach (simpler)**:
- Charge whenever battery drops below 50%
- Works for most configurations
- May waste time on unnecessary charges

**Optimal Approach (efficient)**:
1. Calculate distance field from all objectives
2. Find the maximum distance between any objective and nearest charger
3. If max_battery > max_distance + safety_buffer, minimize charging stops
4. Plan route to pass through chargers naturally, not as dedicated stops

**Example Optimal Route**:
```
Start(7,7) battery=20
→ Park(9,4) battery=15 (don't charge, enough for next leg)
→ Park(12,1) battery=5 (low, but Charger(11,4) is 3 moves away)
→ Charger(11,4) battery=20 (charge here)
→ Park(1,1) battery=7 (don't charge, Charger(2,4) is 4 moves away)
→ Charger(2,4) battery=20 (charge here)
→ Park(4,4) battery=17 (no more objectives, don't charge)
→ WIN
```

### Battery Buffer Zones

**Define safety zones based on distance to chargers**:
- **Green Zone**: ≤5 moves to charger (operate freely)
- **Yellow Zone**: 6-10 moves to charger (monitor battery, maintain 50%+)
- **Orange Zone**: 11-15 moves to charger (maintain 75%+ battery)
- **Red Zone**: 16+ moves to charger (charge before entering)

## Obstacle Pattern Recognition

### Common Maze Patterns

**Pattern 1: Vertical Mazes**
- Characteristics: Buildings form vertical walls, horizontal gaps
- Navigation: Use horizontal corridors, switch columns at gaps
- Example: Columns 1-3 blocked, gap at row 4, columns 5-7 blocked

**Pattern 2: Horizontal Mazes**
- Characteristics: Buildings form horizontal walls, vertical gaps
- Navigation: Use vertical corridors, switch rows at gaps
- Example: Rows 1-3 blocked, gap at column 6, rows 5-7 blocked

**Pattern 3: Island Configurations**
- Characteristics: Water surrounds central area, limited crossings
- Navigation: Identify crossing points, use them efficiently
- Example: Water at rows 6-8, crossings only at columns 4 and 9

**Pattern 4: Scattered Clusters**
- Characteristics: Small building groups, many paths available
- Navigation: Flexible routing, optimize for distance
- Example: 3x3 building clusters with 2-3 cell gaps between

### Bypass Techniques by Pattern

**For Vertical Mazes**:
1. Find horizontal corridor with most vertical gaps
2. Use that corridor as "main highway"
3. Make vertical excursions through gaps to reach objectives

**For Horizontal Mazes**:
1. Find vertical corridor with most horizontal gaps
2. Use that corridor as "main highway"
3. Make horizontal excursions through gaps to reach objectives

**For Island Configurations**:
1. Map all water crossings precisely
2. Plan routes that minimize crossing usage
3. Complete all objectives on one side before crossing to other side

**For Scattered Clusters**:
1. Use straight-line paths where possible
2. Navigate around clusters using Manhattan distance
3. Don't overthink - multiple valid paths exist

## Multi-Section Route Planning

### Grid Sectioning Strategy

**For grids 14x14 or larger**:

1. **Divide into quadrants**:
   - NW: rows 0-6, columns 0-6
   - NE: rows 0-6, columns 7-13
   - SW: rows 7-13, columns 0-6
   - SE: rows 7-13, columns 7-13

2. **Count objectives per quadrant**:
   - NW: 1 park
   - NE: 2 parks
   - SW: 0 parks
   - SE: 1 park

3. **Plan quadrant completion order**:
   - Start in starting quadrant
   - Move to adjacent quadrant with most objectives
   - Complete all objectives in that quadrant
   - Move to next adjacent quadrant
   - Repeat until all quadrants cleared

### Section Transition Optimization

**Goal**: Minimize battery cost when moving between sections.

**Technique: Hub-Based Routing**:
1. Identify "hub" positions at section boundaries
2. Hubs should be on golden corridors in both sections
3. Route through hubs when transitioning sections

**Example**:
- Hub 1: (6,7) - intersection of horizontal row 7 and vertical column 6
- Hub 2: (7,7) - starting position, well-connected
- Route: Quadrant NE → Hub 2 → Quadrant SW → Hub 1 → Quadrant NW

### Backtracking Avoidance

**Rule**: Never leave a section with uncollected objectives unless forced by battery constraints.

**Process**:
1. Enter section
2. Identify all objectives in section
3. Plan route visiting all objectives before exiting
4. Execute planned route
5. Exit to next section

**Exception**: If battery insufficient to collect all objectives and exit, collect subset, recharge, return to section.

## Recovery from Failed Routes

### Failure Analysis Framework

**When a planned route fails**:

1. **Identify failure type**:
   - Obstacle collision: Misread grid or incorrect path
   - Battery depletion: Miscalculated distance or missed charger
   - Wrong objective: Navigation error or incorrect coordinates

2. **Diagnose root cause**:
   - **For obstacles**: Use describe_cell on collision point, verify character type
   - **For battery**: Recalculate distance to nearest charger, check battery math
   - **For navigation**: Verify current position matches expected position

3. **Update mental model**:
   - Correct any grid misinterpretations
   - Add newly discovered obstacles to map
   - Revise distance estimates if off

4. **Plan recovery route**:
   - Assess current battery level
   - Calculate distance to nearest charger if low
   - Find alternative route to original objective or revised objective

### Adaptive Re-Planning

**Technique: Progressive Path Refinement**

Instead of scrapping entire plan after one failure:

1. **Keep successful segments**: Routes that worked remain valid
2. **Replace failed segment only**: Find alternative for blocked section
3. **Adjust downstream route**: Update subsequent steps based on new position

**Example**:
```
Original Plan:
  A → B → C → D → E (failed at B→C, obstacle at C)

Revised Plan:
  A → B ✓ (keep, already completed)
  B → C ✗ (failed, obstacle)
  B → F → C (new alternative segment)
  C → D → E (keep, still valid)
```

### Emergency Battery Management

**When battery critically low (<5) and far from charger**:

1. **Stop all objective collection**
2. **Calculate exact distance to every charger**
3. **Move directly to nearest charger, no detours**
4. **Use single moves, verify each step**
5. **If impossible to reach any charger: Game Over, reset and apply learnings**

**Prevention**: This should never happen with proper planning. If it does, the distance field calculation was wrong or ignored.

## Optimization Goals

**Metrics for Mastery**:
- **Move Efficiency**: Total moves / theoretical minimum moves < 1.5
- **Charging Efficiency**: Charging stops ≤ (total parks / average distance per charge)
- **Success Rate**: Win rate > 90% on first attempt for known configurations
- **Adaptation Speed**: Recovery from failed route within 5 moves

**These metrics are goals for advanced play, not requirements for victory.** The primary goal is systematic completion with learning and improvement.
