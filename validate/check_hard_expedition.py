#!/usr/bin/env python3

import json

# Load the hard_expedition.json configuration
with open('../configs/hard_expedition.json', 'r') as f:
    config = json.load(f)

layout = config['layout']
grid_size = len(layout)

print("Hard Expedition Layout Analysis")
print("=" * 40)
print("Grid (P=park, H=home, S=supercharger, R=road, B=building):")
print()

# Print grid with coordinates
for y, row in enumerate(layout):
    print(f"{y:2d}: {row}")

print()
print("Park locations and connectivity:")
print("-" * 30)

parks = []
homes = []
superchargers = []

# Find all important locations
for y, row in enumerate(layout):
    for x, cell in enumerate(row):
        if cell == 'P':
            parks.append((x, y))
        elif cell == 'H':
            homes.append((x, y))
        elif cell == 'S':
            superchargers.append((x, y))

print(f"Found {len(parks)} parks:")
for i, (x, y) in enumerate(parks):
    print(f"  Park {i+1} at ({x},{y})")
    
    # Check if park is surrounded by buildings
    neighbors = []
    for dx, dy in [(-1,0), (1,0), (0,-1), (0,1)]:
        nx, ny = x + dx, y + dy
        if 0 <= nx < grid_size and 0 <= ny < grid_size:
            neighbors.append(layout[ny][nx])
        else:
            neighbors.append('B')  # Out of bounds = building
    
    passable_neighbors = sum(1 for n in neighbors if n in ['R', 'P', 'H', 'S'])
    print(f"    Neighbors: {neighbors} → {passable_neighbors} passable")
    
    if passable_neighbors == 0:
        print(f"    ❌ UNREACHABLE - completely surrounded by buildings!")
    elif passable_neighbors < 2:
        print(f"    ⚠️  Only 1 exit - potential dead end")
    else:
        print(f"    ✅ Accessible")

print()
print(f"Home locations: {len(homes)}")
for x, y in homes:
    print(f"  Home at ({x},{y})")

print()
print(f"Supercharger locations: {len(superchargers)}")
for x, y in superchargers:
    print(f"  Supercharger at ({x},{y})")

print()
print("CONNECTIVITY ANALYSIS:")
print("=" * 40)

# Simple flood fill to check if all parks are reachable from starting position
# Assuming typical starting position is near a home
if homes:
    start_x, start_y = homes[0]  # Start from first home
    print(f"Testing connectivity from starting point ({start_x},{start_y})")
    
    # Flood fill
    visited = set()
    queue = [(start_x, start_y)]
    
    while queue:
        x, y = queue.pop(0)
        if (x, y) in visited:
            continue
        visited.add((x, y))
        
        # Check all 4 directions
        for dx, dy in [(-1,0), (1,0), (0,-1), (0,1)]:
            nx, ny = x + dx, y + dy
            if (0 <= nx < grid_size and 0 <= ny < grid_size and 
                (nx, ny) not in visited and
                layout[ny][nx] in ['R', 'P', 'H', 'S']):
                queue.append((nx, ny))
    
    # Check if all parks are reachable
    reachable_parks = sum(1 for px, py in parks if (px, py) in visited)
    print(f"Reachable parks: {reachable_parks}/{len(parks)}")
    
    if reachable_parks == len(parks):
        print("✅ All parks are reachable!")
    else:
        print("❌ Some parks are NOT reachable!")
        for i, (px, py) in enumerate(parks):
            status = "✅" if (px, py) in visited else "❌"
            print(f"  {status} Park {i+1} at ({px},{py})")
else:
    print("❌ No home locations found - can't determine starting point")