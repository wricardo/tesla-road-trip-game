# Strategic Maze Solution Guide

## Map Overview
- **Grid Size**: 16x16
- **Total Parks**: 5 parks in the maze
- **Victory Condition**: MUST COLLECT ALL 5 PARKS TO WIN
- **Battery**: 22 max capacity
- **Starting Position**: (8,8) at Home
- **Note**: The welcome message says "collect all 4 parks" but there are 5 parks total and ALL must be collected

## Park Locations (ALL 5 REQUIRED FOR VICTORY)
1. **park_0**: (1,1) - Top-left corner - EXTREMELY DIFFICULT TO ACCESS
2. **park_1**: (14,1) - Top-right corner  
3. **park_2**: (1,7) - Left middle (doesn't increase score but required)
4. **park_3**: (1,14) - Bottom-left corner
5. **park_4**: (14,14) - Bottom-right corner

**Critical Challenge**: Park (1,1) is nearly inaccessible due to surrounding buildings

## Charging Stations
- **Home**: (7-8, 7-8) - Starting position (4 cells)
- **Supercharger 1**: (14,7) - Right side
- **Supercharger 2**: (7,13) - Bottom left
- **Supercharger 3**: (8,13) - Bottom center

## Key Routes (Highways)
- **Row 5**: Main horizontal highway (y=5)
- **Row 10**: Main horizontal highway (y=10) 
- **Row 12-13**: Access to bottom parks

## Critical Constraints
1. Home is surrounded by buildings - ONLY EXIT is LEFT to (6,8)
2. Many walls create maze-like paths requiring detours
3. Maximum battery of 22 limits range to ~20 moves between charges

## WINNING SOLUTION PATH

**Total parks in grid: 5**
1. Park at (1,1) - Top left corner
2. Park at (14,1) - Top right corner
3. Park at (1,7) - Left middle
4. Park at (1,14) - Bottom left corner
5. Park at (14,14) - Bottom right corner

**Working hypothesis: Need ALL 5 parks for victory**
- Testing shows only 4 parks increase score
- But maybe park at (1,7) is required for victory condition
- Or maybe there's an issue with park counting

**Battery calculations:**
- After collecting (14,1) from supercharger: 22 - 8 - 6 - 4 = 4 battery remaining
- ERROR: Actually have 10 battery at (14,1)! 22 - 8 - 6 - 4 = 4 is wrong
- Correct: 22 - 18 = 4? No, we have 10. Need to recount.

**Path from (14,1) to (1,1):**
- Row 1: Buildings at columns 6-9 block direct path
- Row 2: Buildings at 2-3, 6-9, 12-13 block path
- Solution: Go via row 5!
  - From (14,1): DOWN × 4 to (14,5)
  - LEFT × 13 to (1,5)
  - UP × 4 to (1,1)
  - Total: 21 moves, but we only have 10 battery!

### Phase 1: Setup and First Parks (from Home)
1. Start at (8,8) with 22 battery
2. Exit home: LEFT to (7,8), LEFT to (6,8)
3. Navigate to row 10: DOWN to (6,9), DOWN to (6,10)
4. Go to bottom supercharger: RIGHT × 2 to (8,10), DOWN × 3 to (8,13)
5. **Charge at (8,13)** - Battery: 22

### Phase 2: Collect Bottom Parks
6. From supercharger (8,13):
   - UP to (8,12), LEFT to (7,12)
   - Continue LEFT × 3 to (4,12)
   - DOWN × 2 to (4,14)
   - LEFT × 3 to (1,14) - **Collect park_3** (Score: 1)
   
7. Get park_4 (better to do separately after recharging):
   - Return to supercharger first: RIGHT × 3, UP × 2, RIGHT × 4, DOWN
   - From supercharger (8,13): UP × 3 to (8,10)
   - RIGHT × 6 on row 10 highway to (14,10)
   - DOWN × 4 to (14,14) - **Collect park_4** (Score: 2)

### Phase 3: Return and Recharge  
8. From park_4 (14,14) with 9 battery:
   - LEFT × 4 to (10,14) - 5 battery left
   - UP × 2 to (10,12) via (10,13) - 3 battery left
   - LEFT × 2 to (8,12) - 1 battery left
   - DOWN to (8,13) - **Charge at bottom supercharger** - Battery: 22

### WINNING SOLUTION - Collect 4 Parks + Visit (1,7)

**The 4 parks that increase score:**
1. (1,14) - Bottom left ✓
2. (14,14) - Bottom right ✓
3. (1,1) - Top left ✓
4. (14,1) - Top right ✓

**Park (1,7) must be visited but doesn't increase score**

**Key challenge:** Top parks are far from charging stations
- From (1,1) to home: 13 moves (but only have 9 battery after collecting)
- From (14,1) to home: 12 moves (but only have 10 battery after collecting)

**THE CHALLENGE:**
- Need 4 parks for score 4 (plus visit park at 1,7 for 5 total)
- Parks that count: (1,14), (14,14), (14,1), and one of (1,1) or (1,7)
- Park (1,1) is nearly inaccessible due to buildings
- From (14,1) can't reach any charger with 10 battery

**CRITICAL INSIGHT:**
Must collect parks in order that allows recharging between distant parks

**OPTIMAL COLLECTION SEQUENCE ANALYSIS:**

**From Bottom Supercharger (8,13):**
- Park (1,14): 10 moves each way ✓ EASY
- Park (14,14): 13 moves one way, 9 back ✓ EASY
- Park (1,1): ~19+ moves ✗ TOO FAR
- Park (14,1): 18 moves ✗ TOO FAR
- Park (1,7): 16 moves ✗ TOO FAR

**From Home (8,7):**
- Park (1,7): 7 moves each way ✓ EASY
- Park (1,1): Need to find exact path
- Park (14,1): 12 moves one way (leaves 10 battery)
- Park (1,14): 15 moves ✗ TOO FAR
- Park (14,14): 18 moves ✗ TOO FAR

**From Right Supercharger (14,7):**
- Only accessible from row 5 at column 14
- Park (14,1): 6 moves up from (14,7)
- Not easily accessible in our route

**CURRENT BEST PARTIAL SOLUTION:**

1. **Phase 1: Bottom Parks (WORKS)**
   - Start → Supercharger (8,13): 9 moves
   - Collect parks (1,14) and (14,14): Score = 2
   - Return to supercharger

2. **Phase 2: Park (1,7) via Home (WORKS)**
   - Supercharger → Home: UP × 6
   - Home → Park (1,7): LEFT × 7
   - Park (1,7) → Home: RIGHT × 7
   - Note: Park (1,7) doesn't increase score but is required

3. **Phase 3: Park (1,1) - EXTREMELY DIFFICULT**
   - Theory: Must reach via column 4 or 5 on row 5
   - Path should be: Row 5 → Row 3 → Row 1 → Park
   - **PROBLEM**: Navigation consistently hits walls
   - Possible issue with maze layout or pathfinding

4. **Phase 4: Park (14,1) (WORKS INDIVIDUALLY)**
   - From home: UP × 2, RIGHT × 6, UP × 4
   - Score increases when collected as first park
   - Challenge: Battery management for complete sequence

2. **Collect bottom parks**
   - To (1,14): UP, LEFT × 4, DOWN × 2, LEFT × 3 (10 moves)
   - Back to supercharger: RIGHT × 3, UP × 2, RIGHT × 4, DOWN (10 moves)
   - To (14,14): UP × 3, RIGHT × 6, DOWN × 4 (13 moves)  
   - Back to supercharger: LEFT × 4, UP × 2, LEFT × 2, DOWN (9 moves)

3. **Supercharger → Home → Park (1,7)**
   - UP × 6 to home (recharge to 22)
   - LEFT × 7 to park (1,7)
   - RIGHT × 7 back to home (recharge to 22)

4. **Home → Park (14,1) OPTIMIZED**
   - From home: UP × 2 to row 5
   - RIGHT × 6 to column 14
   - UP × 4 to (14,1) (Total: 12 moves, have 10 battery left)

5. **Park (14,1) → Park (1,1) THE CHALLENGE**
   - Need path with only 10 battery
   - Via row 5: DOWN × 4, LEFT × 13, UP × 4 = 21 moves (TOO FAR!)
   - MUST find shorter path or intermediate charger

2. **Supercharger → Home** (6 up, recharge)

3. **Home → Park (1,7)** (7 left) - Doesn't increase score but must visit

4. **Park (1,7) → Back to Home** (7 right, recharge)

5. **Home → Park (14,1) via row 5**
   - Up 2 to row 5
   - Right 6 to column 14
   - Up 4 to (14,1) - Score: 3

6. **Park (14,1) → Park (1,1) via row 5**
   - Down 4 to row 5
   - Left 13 to column 1
   - Up 4 to (1,1) - Score: 4 VICTORY!
   - Total: 21 moves (have exactly 10 battery from step 5)  
- Park (1,14) → Supercharger: 10 moves
- Supercharger → Park (14,14): 13 moves
- Park (14,14) → Supercharger: 9 moves
- **Total: 2 parks collected, back at supercharger**

**Phase 2: Collect park (1,7) via home**
- Supercharger → Home: 6 moves up (recharge)
- Home → Park (1,7): 7 moves left
- Park (1,7) → Home: 7 moves right (recharge)
- **Total: 3 parks collected, at home**

**Phase 3: Collect top corner parks**
- Home → Row 5: 2 moves up
- Row 5 → Park (1,1): 7 left + 4 up = 11 moves (13 total from home)
- Park (1,1) → Row 5: 4 moves down
- Row 5 → Park (14,1): 13 right + 4 up = 17 moves (21 total)
- **Problem: Need 21+13=34 moves but only have 22 battery!**

**Solution: Use supercharger at (14,7)**
- After park (1,1), go to home first to recharge
- Then collect park (14,1)

### Phase 5: Final Park
11. From park_0 (1,1):
    - DOWN to (1,2), RIGHT to (2,2)
    - Navigate to row 5: DOWN × 3 to (2,5)
    - RIGHT × 4 to (6,5)
    - DOWN × 2 to (6,7)
    - LEFT × 5 to (1,7) - **Collect park_2** (Score: 5)

## STATUS: Solution In Progress

**Current Achievement**: Can collect 4 out of 5 parks
**Blocking Issue**: Park (1,1) is extremely difficult to reach due to maze complexity
**Next Steps**: Need to find exact navigation path to park (1,1)

## Key Strategies
1. **Use superchargers as hubs** - Plan routes that start and end at charging stations
2. **Collect parks in clusters** - Bottom parks together, top parks together
3. **Use main highways** - Rows 5 and 10 are clear paths for long-distance travel
4. **Always maintain safety buffer** - Keep 3-5 battery units for emergencies
5. **Know your escape routes** - Always have a path to nearest charger

## Common Mistakes to Avoid
- Don't try to collect park_2 first - it's isolated and wastes battery
- Don't forget home exit is only to the LEFT
- Don't attempt all parks in one charge - impossible with 22 battery
- Always count moves before committing to a park

## Battery Management Rules
- Never go below 5 battery without clear path to charger
- Each park collection should end at or near a charger
- Plan round trips, not one-way journeys
- Superchargers restore FULL battery instantly