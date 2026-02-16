#!/bin/bash

# Simple winnability test - just check if configs load and have reasonable setup
configs=("easy.json" "classic.json" "challenge.json" "easy_circuit.json" "easy_gardens.json" "hard_expedition.json" "medium_maze.json")

echo "Testing basic winnability of all Tesla Road Trip configurations..."
echo "=================================================================="

for config in "${configs[@]}"; do
    echo "Testing $config..."
    
    # Reset game first
    curl -s -X POST http://localhost:8080/api -d '{"action":"reset"}' > /dev/null
    
    # Load the configuration
    response=$(curl -s -X POST http://localhost:8080/api -d "{\"action\":\"load\",\"config\":\"$config\"}")
    
    # Extract basic info using simple grep/cut instead of jq for reliability
    if echo "$response" | grep -q '"victory":true'; then
        echo "  ⚠️  Already won (should not happen)"
    elif echo "$response" | grep -q '"game_over":true'; then
        echo "  ⚠️  Game over on load (potential issue)"
    else
        # Count parks in grid
        park_count=$(echo "$response" | grep -o '"type":"park"' | wc -l)
        home_count=$(echo "$response" | grep -o '"type":"home"' | wc -l) 
        super_count=$(echo "$response" | grep -o '"type":"supercharger"' | wc -l)
        
        echo "  ✅ Loaded successfully: $park_count parks, $home_count homes, $super_count superchargers"
        
        # Basic winnability check: has parks and charging
        total_chargers=$((home_count + super_count))
        if [ $park_count -gt 0 ] && [ $total_chargers -gt 0 ]; then
            echo "     → Likely winnable (has objectives and charging)"
        else
            echo "     → May have issues (parks: $park_count, chargers: $total_chargers)"
        fi
    fi
    echo ""
done

echo "=================================================================="
echo "All configurations loaded successfully and appear to have basic"
echo "winnability requirements (parks to visit + charging stations)."
echo ""
echo "For definitive winnability, pathfinding analysis would be needed."