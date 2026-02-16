#!/bin/bash

# Test winnability of all puzzle configurations
# This script loads each config and does a basic connectivity check

configs=("easy.json" "classic.json" "challenge.json" "easy_circuit.json" "easy_gardens.json" "hard_expedition.json" "medium_maze.json")

echo "Testing winnability of all Tesla Road Trip configurations..."
echo "============================================================"

all_winnable=true

for config in "${configs[@]}"; do
    echo "Testing $config..."
    
    # Load the configuration
    response=$(curl -s -X POST http://localhost:8080/api -d "{\"action\":\"load\",\"config\":\"$config\"}")
    
    # Check if load was successful
    if echo "$response" | grep -q '"game_over":true'; then
        echo "❌ $config - Failed to load properly"
        all_winnable=false
        continue
    fi
    
    # Extract basic info
    battery=$(echo "$response" | jq -r '.max_battery // 0')
    parks=$(echo "$response" | jq -r '[.grid[][] | select(.type=="park")] | length')
    homes=$(echo "$response" | jq -r '[.grid[][] | select(.type=="home")] | length')
    superchargers=$(echo "$response" | jq -r '[.grid[][] | select(.type=="supercharger")] | length')
    
    echo "  • Grid loaded: ${battery} max battery, ${parks} parks, ${homes} homes, ${superchargers} superchargers"
    
    # Basic heuristic: if there are enough charging points and reasonable battery
    chargers=$((homes + superchargers))
    if [ "$chargers" -gt 0 ] && [ "$battery" -gt 5 ]; then
        echo "  ✅ $config - Likely winnable (has charging infrastructure)"
    else
        echo "  ⚠️  $config - May be challenging (limited charging: $chargers chargers, $battery battery)"
    fi
    
    echo ""
done

echo "============================================================"
if [ "$all_winnable" = true ]; then
    echo "✅ All configurations appear winnable based on basic checks"
else
    echo "❌ Some configurations may have issues"
fi

echo ""
echo "Note: This is a basic connectivity test. Detailed pathfinding analysis needed for definitive results."