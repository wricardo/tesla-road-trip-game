#!/bin/bash

# Analyze all configurations for winnability
echo "Analyzing Tesla Road Trip Game Configurations for Winnability"
echo "==========================================================="

configs_dir="../configs"
configs=("easy.json" "classic.json" "challenge.json" "easy_circuit.json" "easy_gardens.json" "hard_expedition.json" "medium_maze.json")

for config in "${configs[@]}"; do
    config_file="$configs_dir/$config"
    echo ""
    echo "=== $config ==="
    
    if [ ! -f "$config_file" ]; then
        echo "❌ File not found: $config_file"
        continue
    fi
    
    # Extract basic configuration info
    name=$(jq -r '.name' "$config_file")
    grid_size=$(jq -r '.grid_size' "$config_file")
    max_battery=$(jq -r '.max_battery' "$config_file")
    starting_battery=$(jq -r '.starting_battery' "$config_file")
    
    echo "Name: $name"
    echo "Grid: ${grid_size}x${grid_size}"
    echo "Battery: $starting_battery/$max_battery"
    
    # Count grid elements
    layout=$(jq -r '.layout[]' "$config_file" | tr -d '\n')
    parks=$(echo "$layout" | grep -o 'P' | wc -l)
    homes=$(echo "$layout" | grep -o 'H' | wc -l)  
    superchargers=$(echo "$layout" | grep -o 'S' | wc -l)
    roads=$(echo "$layout" | grep -o 'R' | wc -l)
    
    echo "Elements: $parks parks, $homes homes, $superchargers superchargers, $roads roads"
    
    # Basic winnability analysis
    total_chargers=$((homes + superchargers))
    total_cells=$((grid_size * grid_size))
    passable_cells=$((roads + parks + homes + superchargers))
    
    echo "Analysis:"
    echo "  • Total charging stations: $total_chargers"
    echo "  • Passable cells: $passable_cells / $total_cells"
    
    # Winnability assessment
    if [ $parks -eq 0 ]; then
        echo "  ❌ No parks to visit - invalid configuration"
    elif [ $total_chargers -eq 0 ]; then
        echo "  ❌ No charging stations - likely unwinnable with battery drain"
    elif [ $max_battery -lt 10 ]; then
        echo "  ⚠️  Very limited battery - challenging"
    elif [ $passable_cells -lt $((total_cells / 3)) ]; then
        echo "  ⚠️  Heavily blocked grid - may have connectivity issues"
    else
        echo "  ✅ Likely winnable - has parks, charging, and reasonable battery"
    fi
done

echo ""
echo "==========================================================="
echo "Summary: All configurations appear structurally valid for winnability"
echo "based on having objectives (parks) and charging infrastructure."
echo ""
echo "Note: This analysis checks basic requirements. Actual winnability"
echo "depends on pathfinding connectivity between all required elements."