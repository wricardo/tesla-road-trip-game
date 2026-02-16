// Tesla Road Trip Game - Main JavaScript
let ws = null;
let gameState = null;
let reconnectTimer = null;
let animationQueue = [];
let isProcessingAnimation = false;
let caveMode = false;
let caveRadius = 2;
let previousVisibleCells = new Set();
let currentSessionId = '';

// Multi-session support
let viewMode = 'single'; // 'single' or 'multi'
let unifiedSessionData = null;
let sessionWebSockets = new Map(); // sessionId -> WebSocket
let sessionColors = ['#e31d23', '#2196f3', '#4caf50', '#ff9800', '#9c27b0', '#00bcd4'];
let sessionIcons = ['🚗', '🚙', '🏎️', '🚘', '🚖', '🚓'];
let activeSessions = new Map(); // sessionId -> {color, icon, data}
let multiSessionRefreshTimer = null;
let sessionListRefreshTimer = null;
let pendingReconnections = new Map(); // Track pending reconnection timers
let multiSelectedSessions = new Set(); // For checkbox selection

// Multi-session animation system
let multiSessionAnimationQueues = new Map(); // sessionId -> queue of states
let multiSessionProcessingAnimation = new Map(); // sessionId -> boolean
let carElements = new Map(); // sessionId -> DOM element
let lastAnimatedMoveIndex = new Map(); // sessionId -> last animated move index

const cellIcons = {
    'home': '🏠',
    'park': '🌳',
    'supercharger': '⚡',
    'water': '💧',
    'building': '🏢',
    'road': '',
    'player': '🚗'
};

// Cave Mode Functions
function calculateVisibleCells(playerX, playerY, radius) {
    const visibleCells = new Set();
    
    for (let y = playerY - radius; y <= playerY + radius; y++) {
        for (let x = playerX - radius; x <= playerX + radius; x++) {
            // Calculate distance using Chebyshev distance (chess-like movement)
            const distance = Math.max(Math.abs(x - playerX), Math.abs(y - playerY));
            
            if (distance <= radius) {
                visibleCells.add(`${x},${y}`);
            }
        }
    }
    
    return visibleCells;
}

function applyCaveMode() {
    if (!gameState || !caveMode) {
        // Remove cave mode if disabled
        document.body.classList.remove('cave-mode');
        document.querySelectorAll('.cave-mode-hidden, .cave-mode-visible, .cave-mode-revealed').forEach(cell => {
            cell.classList.remove('cave-mode-hidden', 'cave-mode-visible', 'cave-mode-revealed');
        });
        return;
    }

    document.body.classList.add('cave-mode');
    
    const visibleCells = calculateVisibleCells(
        gameState.player_pos.x, 
        gameState.player_pos.y, 
        caveRadius
    );
    
    // Apply visibility classes to all cells
    const gridCells = document.querySelectorAll('#grid td');
    gridCells.forEach((cell, index) => {
        const gridSize = gameState.grid.length;
        const x = index % gridSize;
        const y = Math.floor(index / gridSize);
        const cellKey = `${x},${y}`;
        
        // Remove all cave mode classes first
        cell.classList.remove('cave-mode-hidden', 'cave-mode-visible', 'cave-mode-revealed');
        
        if (visibleCells.has(cellKey)) {
            cell.classList.add('cave-mode-visible');
            
            // Add reveal animation for newly visible cells
            if (!previousVisibleCells.has(cellKey)) {
                cell.classList.add('cave-mode-revealed');
                // Remove reveal animation after it completes
                setTimeout(() => {
                    cell.classList.remove('cave-mode-revealed');
                }, 800);
            }
        } else {
            cell.classList.add('cave-mode-hidden');
        }
    });
    
    // Update previous visible cells for next comparison
    previousVisibleCells = new Set(visibleCells);
}

function toggleCaveMode() {
    caveMode = document.getElementById('cave-mode-toggle').checked;
    
    // Save to localStorage
    localStorage.setItem('caveModeEnabled', caveMode);
    
    // Apply immediately if game state exists
    if (gameState) {
        applyCaveMode();
    }
    
    console.log('Cave Mode toggled:', caveMode);
}

function updateRadius(newRadius) {
    caveRadius = parseInt(newRadius);
    document.getElementById('radius-value').textContent = caveRadius;
    
    // Save to localStorage
    localStorage.setItem('caveModeRadius', caveRadius);
    
    // Apply immediately if cave mode is enabled
    if (caveMode && gameState) {
        applyCaveMode();
    }
    
    console.log('Cave Mode radius updated:', caveRadius);
}

function loadCaveModeSettings() {
    // Load settings from localStorage
    const savedCaveMode = localStorage.getItem('caveModeEnabled');
    const savedRadius = localStorage.getItem('caveModeRadius');
    
    if (savedCaveMode !== null) {
        caveMode = savedCaveMode === 'true';
        document.getElementById('cave-mode-toggle').checked = caveMode;
    }
    
    if (savedRadius !== null) {
        caveRadius = parseInt(savedRadius);
        document.getElementById('cave-radius').value = caveRadius;
        document.getElementById('radius-value').textContent = caveRadius;
    }
    
    console.log('Cave Mode settings loaded - enabled:', caveMode, 'radius:', caveRadius);
}

function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const sessionParam = currentSessionId ? `?session=${currentSessionId}` : '';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws${sessionParam}`);

    ws.onopen = () => {
        console.log('WebSocket connected');
        document.getElementById('status').textContent = 'Connected';
        document.getElementById('status').className = 'connection-status connected';
        document.getElementById('message').textContent = 'Connected - Waiting for game state...';
        
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }
    };

    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            
            // Check if this is the new format with session_id
            if (data.session_id && data.game_state) {
                // New format - check if this is for our session
                if (data.session_id === currentSessionId) {
                    console.log('WebSocket received state for our session:', {
                        session_id: data.session_id,
                        player_pos: data.game_state.player_pos,
                        move_history_length: data.game_state.move_history ? data.game_state.move_history.length : 0
                    });
                    queueAnimation(data.game_state);
                }
            } else {
                // Legacy format (backward compatibility)
                console.log('WebSocket received state (legacy format):', {
                    player_pos: data.player_pos,
                    move_history_length: data.move_history ? data.move_history.length : 0
                });
                queueAnimation(data);
            }
        } catch (e) {
            console.error('Failed to parse game state:', e);
        }
    };

    ws.onclose = () => {
        console.log('WebSocket disconnected');
        document.getElementById('status').textContent = 'Disconnected';
        document.getElementById('status').className = 'connection-status disconnected';
        document.getElementById('message').textContent = 'Connection lost - Reconnecting...';

        // Clear any existing timer before setting new one
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }

        // Reconnect after 2 seconds if still in single view mode
        if (viewMode === 'single') {
            reconnectTimer = setTimeout(connect, 2000);
        }
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
}

function queueAnimation(newState) {
    animationQueue.push(newState);

    // Collapse queue if it gets too long (prevent backlog)
    if (animationQueue.length > 5) {
        const lastState = animationQueue[animationQueue.length - 1];
        animationQueue.length = 0;
        animationQueue.push(lastState);
        console.log('Animation queue collapsed to prevent backlog');
    }

    if (!isProcessingAnimation) {
        processNextAnimation();
    }
}

function processNextAnimation() {
    if (animationQueue.length === 0) {
        isProcessingAnimation = false;
        return;
    }

    isProcessingAnimation = true;
    const nextState = animationQueue.shift();
    
    // Update game state
    gameState = nextState;
    renderGame();
    
    // Add delay before processing next animation
    setTimeout(() => {
        processNextAnimation();
    }, 300); // 300ms delay between animations
}

function renderGame() {
    if (!gameState) return;

    // Update score
    document.getElementById('score').textContent = gameState.score;

    // Update total moves
    document.getElementById('total-moves').textContent = gameState.total_moves || 0;

    // Update move history
    updateMoveHistory(gameState.move_history || []);

    // Update battery
    const batteryPercent = (gameState.battery / gameState.max_battery) * 100;
    const batteryBar = document.getElementById('battery-bar');
    batteryBar.style.width = batteryPercent + '%';
    batteryBar.className = 'battery-fill';
    if (gameState.battery <= 3) {
        batteryBar.classList.add('low');
    } else if (gameState.battery <= 6) {
        batteryBar.classList.add('medium');
    }
    document.getElementById('battery-text').textContent = 
        `${gameState.battery}/${gameState.max_battery}`;

    // Update message
    const messageEl = document.getElementById('message');
    messageEl.textContent = gameState.message || 'Ready';
    messageEl.className = 'message';
    if (gameState.game_over) {
        messageEl.classList.add('error');
    } else if (gameState.victory) {
        messageEl.classList.add('success');
    } else if (gameState.battery <= 3) {
        messageEl.classList.add('warning');
    } else {
        messageEl.classList.add('info');
    }

    // Render grid
    const gridEl = document.getElementById('grid');
    gridEl.innerHTML = '';

    // Pre-compute trail positions using Set for O(1) lookups
    const trailPositions = new Set();
    const recentTrailPositions = new Set();
    if (gameState.move_history) {
        const recentMoves = gameState.move_history.slice(-5); // Last 5 moves
        gameState.move_history.forEach(move => {
            if (move.to_position && move.success) {
                const key = `${move.to_position.x},${move.to_position.y}`;
                trailPositions.add(key);
            }
        });
        recentMoves.forEach(move => {
            if (move.to_position && move.success) {
                const key = `${move.to_position.x},${move.to_position.y}`;
                recentTrailPositions.add(key);
            }
        });
    }

    for (let y = 0; y < gameState.grid.length; y++) {
        const row = gridEl.insertRow();
        for (let x = 0; x < gameState.grid[y].length; x++) {
            const cell = row.insertCell();
            const cellData = gameState.grid[y][x];

            // Set cell class
            cell.className = `cell-${cellData.type}`;
            if (cellData.visited) {
                cell.classList.add('visited');
            }

            // Add content first
            if (gameState.player_pos.x === x && gameState.player_pos.y === y) {
                // Player is here - show crash icon if game over but not victory
                const playerIcon = (gameState.game_over && !gameState.victory) ? '💥' : cellIcons.player;
                cell.innerHTML = `<span class="player">${playerIcon}</span>`;
            } else {
                // Show cell icon
                const icon = cellIcons[cellData.type];
                if (icon) {
                    if (cellData.type === 'park' && cellData.visited) {
                        cell.textContent = '✓';
                    } else {
                        cell.textContent = icon;
                    }
                } else {
                    cell.textContent = '';
                }
            }

            // Add trail dots from move history (after content)
            const posKey = `${x},${y}`;
            if (trailPositions.has(posKey) && !(gameState.player_pos.x === x && gameState.player_pos.y === y)) {
                // Add trail dot if not current player position
                cell.style.position = 'relative';
                const trailDot = document.createElement('div');
                trailDot.className = 'trail-dot';

                // Fade older trail dots
                if (!recentTrailPositions.has(posKey)) {
                    trailDot.classList.add('fade');
                }

                cell.appendChild(trailDot);
            }
        }
    }

    // Always hide the game over overlay - we don't want it anywhere
    document.getElementById('game-over').classList.remove('active');

    // Apply cave mode after grid is rendered
    applyCaveMode();
}

// Copy prompt function
function copyPrompt(event) {
    const systemPrompt = document.getElementById('ai-prompt-content').querySelector('pre').textContent;
    const userTask = document.getElementById('user-task').textContent.trim();

    // Clean up the user task if it still has the placeholder text
    const cleanTask = userTask.includes('[Edit this') ? 'Help me play the Tesla Road Trip game' : userTask;

    // Combine system prompt with user task
    const fullPrompt = systemPrompt.replace('[Task will be inserted here]', cleanTask);

    // Create a temporary textarea to copy from
    const textarea = document.createElement('textarea');
    textarea.value = fullPrompt;
    textarea.style.position = 'absolute';
    textarea.style.left = '-9999px';
    document.body.appendChild(textarea);

    // Select and copy
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);

    // Visual feedback
    const button = event.target;
    const originalText = button.textContent;
    button.textContent = '✅ Copied!';
    button.style.background = '#2196F3';
    setTimeout(() => {
        button.textContent = originalText;
        button.style.background = '#4caf50';
    }, 2000);
}

// Toggle menu function
function toggleMenu() {
    const menu = document.getElementById('slideMenu');
    menu.classList.toggle('open');
}

// Update move history display
function updateMoveHistory(moveHistory) {
    const historyList = document.getElementById('move-history-list');
    
    if (!moveHistory || moveHistory.length === 0) {
        historyList.innerHTML = '<p style="color: #939395; text-align: center; margin: 0; padding: 24px 0; font-weight: 400;">No moves yet</p>';
        return;
    }

    // Show most recent moves first
    const recentMoves = moveHistory.slice(-8).reverse();
    
    let html = '';
    recentMoves.forEach((move, index) => {
        const timestamp = new Date(move.timestamp * 1000).toLocaleTimeString('en-US', { 
            hour: '2-digit', 
            minute: '2-digit', 
            second: '2-digit' 
        });
        const statusColor = move.success ? '#008000' : '#cc0000';
        const direction = move.action.charAt(0).toUpperCase() + move.action.slice(1);
        
        html += `
            <div style="display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid #f4f4f4; background: ${index === 0 ? '#fafafa' : 'transparent'};">
                <span style="font-weight: 500; color: #171a20; letter-spacing: 0;">${direction}</span>
                <span style="color: ${statusColor}; font-size: 13px; font-weight: 400;">${timestamp}</span>
            </div>
        `;
    });
    
    if (moveHistory.length > 8) {
        html += `<div style="color: #939395; text-align: center; padding: 12px; font-size: 13px; background: #fafafa;">+${moveHistory.length - 8} earlier moves</div>`;
    }
    
    historyList.innerHTML = html;
}

// Combined Move History Functions
async function fetchCombinedMoveHistory() {
    const allMoves = [];
    const sessionStats = {
        totalSessions: activeSessions.size,
        totalMoves: 0,
        sessionsWithMoves: 0
    };

    // Fetch move history from all active sessions
    for (const [sessionId, sessionInfo] of activeSessions) {
        try {
            const response = await fetch(`/api/sessions/${sessionId}/history?limit=50&order=desc`);
            if (response.ok) {
                const historyData = await response.json();
                const moves = historyData.moves || [];

                if (moves.length > 0) {
                    sessionStats.sessionsWithMoves++;
                    sessionStats.totalMoves += moves.length;

                    // Add session info to each move
                    moves.forEach(move => {
                        allMoves.push({
                            ...move,
                            sessionId: sessionId,
                            sessionColor: sessionInfo.color,
                            sessionIcon: sessionInfo.icon
                        });
                    });
                }
            }
        } catch (error) {
            console.error(`Failed to fetch history for session ${sessionId}:`, error);
        }
    }

    // Sort all moves by timestamp (newest first)
    allMoves.sort((a, b) => b.timestamp - a.timestamp);

    // Take the most recent moves (limit to prevent overwhelming UI)
    const recentMoves = allMoves.slice(0, 100);

    return { moves: recentMoves, stats: sessionStats };
}

async function updateCombinedMoveHistory() {
    if (viewMode !== 'multi' || activeSessions.size === 0) {
        return;
    }

    const historyContainer = document.getElementById('combinedMoveHistory');
    const historyStats = document.getElementById('historyStats');

    if (!historyContainer || !historyStats) {
        return;
    }

    try {
        const { moves, stats } = await fetchCombinedMoveHistory();

        // Update stats display
        historyStats.textContent = `${stats.totalMoves} moves from ${stats.sessionsWithMoves}/${stats.totalSessions} sessions`;

        if (moves.length === 0) {
            historyContainer.innerHTML = `
                <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 40px; color: #666;">
                    <span>No moves found in active sessions</span>
                </div>
            `;
            return;
        }

        // Group moves by time periods for better readability
        const groupedMoves = groupMovesByTime(moves);

        let html = '';

        for (const [timeGroup, groupMoves] of Object.entries(groupedMoves)) {
            html += `
                <div style="background: #f8f9fa; padding: 8px 12px; border-bottom: 1px solid #e0e0e0; font-size: 13px; font-weight: 500; color: #666; position: sticky; top: 0;">
                    ${timeGroup}
                </div>
            `;

            groupMoves.forEach(move => {
                const timestamp = new Date(move.timestamp * 1000);
                const timeStr = timestamp.toLocaleTimeString('en-US', {
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit'
                });

                const statusColor = move.success ? '#008000' : '#cc0000';
                const direction = move.action.charAt(0).toUpperCase() + move.action.slice(1);

                html += `
                    <div style="display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; border-bottom: 1px solid #f4f4f4; background: white; transition: background-color 0.2s ease;">
                        <div style="display: flex; align-items: center; gap: 8px;">
                            <div style="display: flex; align-items: center; gap: 6px; background: ${move.sessionColor}15; border: 1px solid ${move.sessionColor}40; border-radius: 12px; padding: 3px 8px; min-width: 60px;">
                                <span style="font-size: 14px;">${move.sessionIcon}</span>
                                <span style="font-size: 11px; font-weight: 600; color: ${move.sessionColor}; letter-spacing: 0.5px;">${move.sessionId}</span>
                            </div>
                            <span style="font-weight: 500; color: #171a20; letter-spacing: 0;">${direction}</span>
                            ${!move.success ? '<span style="color: #cc0000; font-size: 12px;">⚠️</span>' : ''}
                        </div>
                        <span style="color: #999; font-size: 12px; font-weight: 400;">${timeStr}</span>
                    </div>
                `;
            });
        }

        historyContainer.innerHTML = html;

        // Auto-scroll to top to show newest moves
        historyContainer.scrollTop = 0;

    } catch (error) {
        console.error('Failed to update combined move history:', error);
        historyContainer.innerHTML = `
            <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 40px; color: #cc0000;">
                <span>Error loading move history</span>
                <button onclick="refreshCombinedHistory()" style="margin-top: 12px; padding: 6px 12px; background: #f44336; color: white; border: none; border-radius: 4px; cursor: pointer;">Retry</button>
            </div>
        `;
        historyStats.textContent = 'Error loading';
    }
}

function groupMovesByTime(moves) {
    const groups = {};
    const now = new Date();

    moves.forEach(move => {
        const moveDate = new Date(move.timestamp * 1000);
        const diffMs = now - moveDate;
        const diffSeconds = Math.floor(diffMs / 1000);
        const diffMinutes = Math.floor(diffSeconds / 60);
        const diffHours = Math.floor(diffMinutes / 60);

        let groupKey;
        if (diffSeconds < 60) {
            groupKey = 'Just now';
        } else if (diffMinutes < 60) {
            groupKey = `${diffMinutes} minute${diffMinutes === 1 ? '' : 's'} ago`;
        } else if (diffHours < 24) {
            groupKey = `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
        } else {
            groupKey = moveDate.toLocaleDateString();
        }

        if (!groups[groupKey]) {
            groups[groupKey] = [];
        }
        groups[groupKey].push(move);
    });

    return groups;
}

async function refreshCombinedHistory() {
    const historyContainer = document.getElementById('combinedMoveHistory');
    const historyStats = document.getElementById('historyStats');

    if (historyContainer) {
        historyContainer.innerHTML = `
            <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 40px; color: #666;">
                <div style="width: 24px; height: 24px; border: 2px solid #f3f3f3; border-top: 2px solid #4caf50; border-radius: 50%; animation: spin 1s linear infinite; margin-bottom: 12px;"></div>
                <span>Refreshing history...</span>
            </div>
        `;
    }

    if (historyStats) {
        historyStats.textContent = 'Refreshing...';
    }

    await updateCombinedMoveHistory();
}

// Function to load and display configurations
async function loadConfigurations() {
    try {
        // First get the list of configs
        const response = await fetch('/api/configs');
        const configList = await response.json();

        // Then fetch full details for each config
        const configsWithDetails = await Promise.all(
            configList.map(async (config) => {
                try {
                    const name = config.filename.replace('.json', '');
                    const detailResponse = await fetch(`/api/configs/${name}`);
                    const details = await detailResponse.json();
                    return {
                        ...config,
                        ...details
                    };
                } catch (err) {
                    console.error(`Failed to load config ${config.filename}:`, err);
                    return config; // Return partial data if detail fetch fails
                }
            })
        );

        displayConfigurations(configsWithDetails);
    } catch (error) {
        console.error('Failed to load configurations:', error);
        document.getElementById('config-list').innerHTML =
            '<p style="color: #cc0000; text-align: center; padding: 20px;">Failed to load configurations</p>';
    }
}

function displayConfigurations(configs) {
    const configList = document.getElementById('config-list');
    
    if (!configs || configs.length === 0) {
        configList.innerHTML = '<p style="color: #666; text-align: center; padding: 20px;">No configurations available</p>';
        return;
    }

    let html = '';
    configs.forEach(config => {
        // Determine difficulty based on battery and grid size
        let difficulty = 'easy';
        if (config.grid_size >= 18 || config.max_battery <= 10) {
            difficulty = 'hard';
        } else if (config.grid_size >= 15 || config.max_battery <= 20) {
            difficulty = 'medium';
        }

        // Count parks in layout
        let parkCount = 0;
        config.layout.forEach(row => {
            for (let char of row) {
                if (char === 'P') parkCount++;
            }
        });

        html += `
            <div class="config-item">
                <div class="config-header" onclick="toggleConfig('${config.filename}')">
                    <div>
                        <div class="config-title">${config.name} <span style="color: #888; font-size: 0.85em; font-weight: normal;">(${config.filename})</span></div>
                        <div style="font-size: 12px; color: #666; margin-top: 4px;">${config.description}</div>
                    </div>
                    <div style="display: flex; gap: 8px; align-items: center;">
                        ${config.is_active ? '<span class="config-badge active">ACTIVE</span>' : ''}
                        <span class="config-badge ${difficulty}">${difficulty.toUpperCase()}</span>
                    </div>
                </div>
                <div class="config-details" id="config-${config.filename}">
                    <div class="config-stats">
                        <div class="config-stat">
                            <span class="config-stat-label">Grid Size</span>
                            <span class="config-stat-value">${config.grid_size}×${config.grid_size}</span>
                        </div>
                        <div class="config-stat">
                            <span class="config-stat-label">Battery</span>
                            <span class="config-stat-value">${config.starting_battery}/${config.max_battery}</span>
                        </div>
                        <div class="config-stat">
                            <span class="config-stat-label">Parks to Collect</span>
                            <span class="config-stat-value">${parkCount}</span>
                        </div>
                        <div class="config-stat">
                            <span class="config-stat-label">Wall Crash</span>
                            <span class="config-stat-value" style="color: ${config.wall_crash_ends_game ? '#cc0000' : '#008000'}">
                                ${config.wall_crash_ends_game ? 'Ends Game' : 'Safe'}
                            </span>
                        </div>
                    </div>
                    <div style="margin-bottom: 8px;">
                        <span style="font-size: 12px; color: #666;">Grid Preview:</span>
                    </div>
                    <div class="config-grid-preview">
                        ${renderMiniGrid(config.layout)}
                    </div>
                </div>
            </div>
        `;
    });

    configList.innerHTML = html;
}

function renderMiniGrid(layout) {
    let html = '<table>';
    layout.forEach(row => {
        html += '<tr>';
        for (let char of row) {
            let cellClass = '';
            let cellContent = '';
            switch(char) {
                case 'R': cellClass = 'preview-road'; break;
                case 'H': cellClass = 'preview-home'; cellContent = '🏠'; break;
                case 'P': cellClass = 'preview-park'; cellContent = '🌳'; break;
                case 'S': cellClass = 'preview-supercharger'; cellContent = '⚡'; break;
                case 'W': cellClass = 'preview-water'; cellContent = '💧'; break;
                case 'B': cellClass = 'preview-building'; break;
            }
            html += `<td class="${cellClass}">${cellContent}</td>`;
        }
        html += '</tr>';
    });
    html += '</table>';
    return html;
}

function toggleConfig(filename) {
    const details = document.getElementById(`config-${filename}`);
    if (details) {
        details.classList.toggle('open');
    }
}

// Multi-session view functions
function setViewMode(mode) {
    console.log(`🎮 View mode button clicked! Switching to: ${mode} mode`);
    console.log(`Current session ID: ${currentSessionId || 'none'}`);
    viewMode = mode;

    // Update button styles
    const singleBtn = document.getElementById('singleViewBtn');
    const multiBtn = document.getElementById('multiViewBtn');
    
    if (mode === 'single') {
        singleBtn.style.background = '#4caf50';
        singleBtn.style.color = 'white';
        multiBtn.style.background = '#e0e0e0';
        multiBtn.style.color = '#333';
        
        // Show single session dashboard
        document.getElementById('gameDashboard').style.display = 'flex';
        document.getElementById('multiSessionDashboard').style.display = 'none';
        
        // Clear car elements but preserve animation state for when we return
        carElements.forEach(el => el.remove());
        carElements.clear();
        // Don't clear animation queues and processing state completely
        // Just pause them while in single view
        
        // Close multi-session WebSockets
        sessionWebSockets.forEach((ws, sessionId) => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.close();
            }
        });
        sessionWebSockets.clear();
    } else {
        singleBtn.style.background = '#e0e0e0';
        singleBtn.style.color = '#333';
        multiBtn.style.background = '#4caf50';
        multiBtn.style.color = 'white';
        
        // Show multi-session dashboard
        document.getElementById('gameDashboard').style.display = 'none';
        document.getElementById('multiSessionDashboard').style.display = 'flex';
        
        // Load session dropdown options
        loadSessionDropdown();
        
        // If we have a current session, auto-load it
        if (currentSessionId) {
            loadUnifiedSessions([currentSessionId]);
        } else if (!unifiedSessionData) {
            loadUnifiedSessions();
        } else {
            // Restart WebSockets if we already have data
            startMultiSessionWebSockets();
        }
    }
}

function loadUnifiedSessions(sessionIds = null) {
    let url = '/api/sessions/unified';
    if (sessionIds && sessionIds.length > 0) {
        url += '?sessionIds=' + sessionIds.join(',');
    } else if (currentSessionId) {
        // Try to load sessions with same config as current session
        fetch(`/api/sessions/${currentSessionId}`)
            .then(response => response.json())
            .then(sessionData => {
                const configName = sessionData.game_config?.name;
                if (configName) {
                    loadUnifiedSessionsByConfig(configName);
                }
            })
            .catch(error => console.error('Failed to get session config:', error));
        return;
    }
    
    fetch(url)
        .then(response => response.json())
        .then(data => {
            unifiedSessionData = data;
            updateUnifiedView();
            startMultiSessionWebSockets();
            // Reload dropdown to update available sessions
            loadSessionDropdown();
        })
        .catch(error => {
            console.error('Failed to load unified sessions:', error);
            document.getElementById('unifiedConfigName').textContent = 'No sessions available';
        });
}

function loadUnifiedSessionsByConfig(configName) {
    fetch(`/api/sessions/unified?configName=${encodeURIComponent(configName)}`)
        .then(response => response.json())
        .then(data => {
            unifiedSessionData = data;
            updateUnifiedView();
            startMultiSessionWebSockets();
        })
        .catch(error => {
            console.error('Failed to load unified sessions:', error);
        });
}

function loadAllSessionsWithConfig() {
    if (unifiedSessionData && unifiedSessionData.config_name) {
        loadUnifiedSessionsByConfig(unifiedSessionData.config_name);
    } else if (currentSessionId) {
        // Get config from current session
        fetch(`/api/sessions/${currentSessionId}`)
            .then(response => response.json())
            .then(sessionData => {
                const configName = sessionData.game_config?.name;
                if (configName) {
                    loadUnifiedSessionsByConfig(configName);
                }
            });
    }
}

function loadSessionDropdown() {
    const select = document.getElementById('addSessionSelect');
    const status = document.getElementById('sessionListStatus');
    
    // Show loading status
    status.style.display = 'flex';
    status.innerHTML = '<span class="session-list-spinner"></span> Loading sessions...';
    
    fetch('/api/sessions')
        .then(response => response.json())
        .then(data => {
            select.innerHTML = '<option value="">Select a session...</option>';
            
            // Get current session IDs in unified view
            const currentIds = unifiedSessionData ?
                unifiedSessionData.sessions.map(s => s.session_id || s.id) : [];
            
            // Check if data has sessions array
            const sessions = data.sessions || [];
            
            sessions.forEach(session => {
                // Skip sessions already in unified view
                if (!currentIds.includes(session.id)) {
                    const option = document.createElement('option');
                    option.value = session.id;
                    option.textContent = `${session.id} - ${session.config_name}`;
                    if (session.is_active) {
                        option.textContent += ' (active)';
                    }
                    select.appendChild(option);
                }
            });
            
            if (select.options.length === 1) {
                status.innerHTML = '<span style="color: #888;">All sessions are already in view</span>';
            } else {
                status.style.display = 'none';
            }
        })
        .catch(error => {
            console.error('Failed to load sessions:', error);
            status.innerHTML = '<span style="color: #cc0000;">Failed to load sessions</span>';
        });
}

function addSelectedSessionToUnified() {
    const select = document.getElementById('addSessionSelect');
    const sessionId = select.value;
    
    if (!sessionId) {
        alert('Please select a session from the dropdown');
        return;
    }
    
    // Add to existing session list or create new list
    const currentIds = unifiedSessionData ? 
        unifiedSessionData.sessions.map(s => s.session_id) : [];
    
    if (!currentIds.includes(sessionId)) {
        currentIds.push(sessionId);
        loadUnifiedSessions(currentIds);
    }
    
    // Reset the dropdown
    select.value = '';
    // Reload the dropdown to update available sessions
    loadSessionDropdown();
}

function startMultiSessionWebSockets() {
    // Close existing WebSocket connections and clean up
    sessionWebSockets.forEach((ws, sessionId) => {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.close();
        }
        // Clean up animation queues for removed sessions
        multiSessionAnimationQueues.delete(sessionId);
        multiSessionProcessingAnimation.delete(sessionId);
    });
    sessionWebSockets.clear();
    
    // Open WebSocket for each session
    if (unifiedSessionData && unifiedSessionData.sessions) {
        unifiedSessionData.sessions.forEach(session => {
            connectSessionWebSocket(session.session_id);
        });
    }
}

function connectSessionWebSocket(sessionId) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws?session=${sessionId}`);
    
    ws.onopen = () => {
        console.log(`WebSocket connected for session ${sessionId}`);
    };
    
    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            
            // Check if this is the new format with session_id
            if (data.session_id && data.game_state) {
                // New format - check if this is for the correct session
                if (data.session_id === sessionId && activeSessions.has(sessionId)) {
                    console.log(`WebSocket update for session ${sessionId}:`, data.game_state);
                    
                    const newData = {
                        session_id: sessionId,
                        game_state: {
                            player_pos: data.game_state.player_pos,
                            battery: data.game_state.battery,
                            max_battery: data.game_state.max_battery,
                            score: data.game_state.score,
                            visited_parks: data.game_state.visited_parks || {},
                            game_over: data.game_state.game_over,
                            victory: data.game_state.victory,
                            config_name: data.game_state.config_name,
                            total_moves: data.game_state.total_moves || 0,
                            move_history: data.game_state.move_history || []
                        }
                    };
                    
                    // Queue the animation for this session
                    queueMultiSessionAnimation(sessionId, newData);
                }
            } else {
                // Legacy format - assume it's for this session
                if (activeSessions.has(sessionId)) {
                    console.log(`WebSocket update for session ${sessionId} (legacy):`, data);
                    
                    const newData = {
                        session_id: sessionId,
                        game_state: {
                            player_pos: data.player_pos,
                            battery: data.battery,
                            max_battery: data.max_battery,
                            score: data.score,
                            visited_parks: data.visited_parks || {},
                            game_over: data.game_over,
                            victory: data.victory,
                            config_name: data.config_name,
                            total_moves: data.total_moves || 0
                        }
                    };
                    
                    // Queue the animation for this session
                    queueMultiSessionAnimation(sessionId, newData);
                }
            }
        } catch (e) {
            console.error(`Failed to parse WebSocket data for session ${sessionId}:`, e);
        }
    };
    
    ws.onclose = () => {
        console.log(`WebSocket disconnected for session ${sessionId}`);
        sessionWebSockets.delete(sessionId);

        // Clear any existing reconnection timer for this session
        if (pendingReconnections.has(sessionId)) {
            clearTimeout(pendingReconnections.get(sessionId));
            pendingReconnections.delete(sessionId);
        }

        // Try to reconnect after 2 seconds if still in multi-view mode
        if (viewMode === 'multi' && activeSessions.has(sessionId)) {
            const timer = setTimeout(() => {
                pendingReconnections.delete(sessionId);
                connectSessionWebSocket(sessionId);
            }, 2000);
            pendingReconnections.set(sessionId, timer);
        }
    };
    
    ws.onerror = (error) => {
        console.error(`WebSocket error for session ${sessionId}:`, error);
    };
    
    sessionWebSockets.set(sessionId, ws);
}

function queueMultiSessionAnimation(sessionId, newData) {
    // Initialize queue if needed
    if (!multiSessionAnimationQueues.has(sessionId)) {
        multiSessionAnimationQueues.set(sessionId, []);
        multiSessionProcessingAnimation.set(sessionId, false);
    }

    // Add to queue
    const queue = multiSessionAnimationQueues.get(sessionId);
    queue.push(newData);

    // Collapse queue if it gets too long (prevent backlog)
    if (queue.length > 5) {
        const lastData = queue[queue.length - 1];
        queue.length = 0;
        queue.push(lastData);
        console.log(`Multi-session animation queue collapsed for session ${sessionId}`);
    }

    // Process if not already processing
    if (!multiSessionProcessingAnimation.get(sessionId)) {
        processMultiSessionAnimation(sessionId);
    }
}

function processMultiSessionAnimation(sessionId) {
    const queue = multiSessionAnimationQueues.get(sessionId);
    
    if (!queue || queue.length === 0) {
        multiSessionProcessingAnimation.set(sessionId, false);
        return;
    }
    
    multiSessionProcessingAnimation.set(sessionId, true);
    const newData = queue.shift();
    
    // Update session data
    const sessionInfo = activeSessions.get(sessionId);
    if (sessionInfo) {
        sessionInfo.data = newData;

        // Re-render the unified grid to update trails
        renderUnifiedGrid();

        // Animate the car movement
        animateCarMovement(sessionId, newData.game_state.player_pos);

        // Update stats and legend
        updateSessionStats();
    }
    
    // Process next animation after delay
    setTimeout(() => {
        processMultiSessionAnimation(sessionId);
    }, 300); // 300ms delay between moves
}

function animateCarMovement(sessionId, newPos) {
    const sessionInfo = activeSessions.get(sessionId);
    if (!sessionInfo || !sessionInfo.data || !sessionInfo.data.game_state) return;

    const moveHistory = sessionInfo.data.game_state.move_history || [];

    // If no move history, fall back to instant positioning
    if (moveHistory.length === 0) {
        positionCarInstantly(sessionId, newPos);
        lastAnimatedMoveIndex.set(sessionId, -1);
        return;
    }

    // Get the last animated move index for this session
    const lastAnimatedIndex = lastAnimatedMoveIndex.get(sessionId) ?? -1;

    // Check if we have new moves to animate
    if (lastAnimatedIndex >= moveHistory.length - 1) {
        // No new moves, just position the car at the current position
        positionCarInstantly(sessionId, newPos);
        return;
    }

    // Calculate the starting index for animation (next move after last animated)
    const startIndex = lastAnimatedIndex + 1;

    // Animate only the new moves
    animateCarThroughMoveHistory(sessionId, moveHistory, startIndex);
}

function positionCarInstantly(sessionId, pos) {
    const gridEl = document.getElementById('unifiedGrid');
    if (!gridEl || !unifiedSessionData) return;

    const sessionInfo = activeSessions.get(sessionId);
    if (!sessionInfo) return;

    // Remove old car element if it exists
    let oldCarEl = carElements.get(sessionId);
    if (oldCarEl) {
        oldCarEl.remove();
    }

    // Find the target cell
    const targetCell = gridEl.rows[pos.y]?.cells[pos.x];
    if (!targetCell) return;

    // Create new car element in the target cell
    const carEl = document.createElement('div');
    carEl.className = 'multi-car';
    carEl.id = `car-${sessionId}`;

    // Use crash icon if game over but not victory
    if (sessionInfo.data.game_state?.game_over && !sessionInfo.data.game_state?.victory) {
        carEl.innerHTML = '💥'; // Crash/explosion icon
        carEl.style.filter = `drop-shadow(0 0 8px #ff4444)`;
    } else {
        carEl.innerHTML = sessionInfo.icon;
        carEl.style.filter = `drop-shadow(0 0 6px ${sessionInfo.color})`;
    }

    targetCell.appendChild(carEl);
    carElements.set(sessionId, carEl);

    updateGridCellForVisitedParks(pos);
}

function animateCarThroughMoveHistory(sessionId, moveHistory, startIndex = 0) {
    const sessionInfo = activeSessions.get(sessionId);
    if (!sessionInfo) return;

    let currentStep = startIndex;

    function animateNextStep() {
        if (currentStep >= moveHistory.length) {
            // Animation complete, update grid for visited parks
            const finalPos = sessionInfo.data.game_state.player_pos;
            updateGridCellForVisitedParks(finalPos);
            // Update the last animated move index
            lastAnimatedMoveIndex.set(sessionId, moveHistory.length - 1);
            return;
        }

        const move = moveHistory[currentStep];
        const targetPos = move.to_position;

        // Position the car at this step's position
        positionCarAtStep(sessionId, targetPos, currentStep === moveHistory.length - 1);

        // Update the last animated move index as we animate
        lastAnimatedMoveIndex.set(sessionId, currentStep);

        currentStep++;

        // Continue with next step after 300ms delay
        setTimeout(animateNextStep, 300);
    }

    // If starting from the beginning, position at first move's from_position
    if (startIndex === 0 && moveHistory.length > 0 && moveHistory[0].from_position) {
        positionCarAtStep(sessionId, moveHistory[0].from_position, false);
    }

    // Start the animation sequence
    animateNextStep();
}

function positionCarAtStep(sessionId, pos, isFinalStep) {
    const gridEl = document.getElementById('unifiedGrid');
    if (!gridEl || !unifiedSessionData) return;

    const sessionInfo = activeSessions.get(sessionId);
    if (!sessionInfo) return;

    // Remove old car element if it exists
    let oldCarEl = carElements.get(sessionId);
    if (oldCarEl) {
        oldCarEl.remove();
    }

    // Find the target cell
    const targetCell = gridEl.rows[pos.y]?.cells[pos.x];
    if (!targetCell) return;

    // Create new car element in the target cell
    const carEl = document.createElement('div');
    carEl.className = 'multi-car';
    carEl.id = `car-${sessionId}`;

    // Use crash icon if this is the final step and game is over but not victory
    if (isFinalStep && sessionInfo.data.game_state?.game_over && !sessionInfo.data.game_state?.victory) {
        carEl.innerHTML = '💥'; // Crash/explosion icon
        carEl.style.filter = `drop-shadow(0 0 8px #ff4444)`;
    } else {
        carEl.innerHTML = sessionInfo.icon;
        carEl.style.filter = `drop-shadow(0 0 6px ${sessionInfo.color})`;
    }

    targetCell.appendChild(carEl);
    carElements.set(sessionId, carEl);
}

function updateGridCellForVisitedParks(pos) {
    const gridEl = document.getElementById('unifiedGrid');
    if (!gridEl) return;

    // Update grid cell classes for visited parks
    const cell = gridEl.rows[pos.y]?.cells[pos.x];
    if (cell && unifiedSessionData.sessions && unifiedSessionData.sessions.length > 0) {
        const firstSession = unifiedSessionData.sessions[0];
        const grid = firstSession.game_state ? firstSession.game_state.grid : null;
        if (grid && grid[pos.y] && grid[pos.y][pos.x]) {
            const cellData = grid[pos.y][pos.x];

            // Check all sessions for visited parks at this position
            for (const session of unifiedSessionData.sessions) {
                if (cellData.type === 'park' && session.game_state &&
                    session.game_state.visited_parks && session.game_state.visited_parks[cellData.id]) {
                    cell.classList.add('visited');
                    break;
                }
            }
        }
    }
}

function updateUnifiedViewWithoutReload() {
    // Legacy function - now uses animation system
    // Update stats and legend only
    updateSessionLegend();
    updateSessionStats();
}

function updateUnifiedView() {
    if (!unifiedSessionData) return;

    // Update config info
    document.getElementById('unifiedConfigName').textContent = unifiedSessionData.config_name || 'Unknown';
    document.getElementById('totalSessionCount').textContent = unifiedSessionData.sessions.length;
    document.getElementById('totalParksCount').textContent = unifiedSessionData.total_parks || 0;

    // Preserve existing animation tracking for sessions that still exist
    const oldAnimationIndexes = new Map(lastAnimatedMoveIndex);

    // Assign colors and icons to sessions
    activeSessions.clear();
    // Don't clear lastAnimatedMoveIndex - preserve it for existing sessions
    unifiedSessionData.sessions.forEach((session, index) => {
        activeSessions.set(session.session_id, {
            color: sessionColors[index % sessionColors.length],
            icon: sessionIcons[index % sessionIcons.length],
            data: session
        });

        // Initialize animation index for new sessions only
        if (!lastAnimatedMoveIndex.has(session.session_id)) {
            // Set to current move count minus 1 to skip replaying old animations
            const moveCount = session.game_state?.move_history?.length || 0;
            lastAnimatedMoveIndex.set(session.session_id, moveCount - 1);
        }
    });

    // Clean up animation tracking for sessions that no longer exist
    for (const [sessionId] of lastAnimatedMoveIndex) {
        if (!activeSessions.has(sessionId)) {
            lastAnimatedMoveIndex.delete(sessionId);
        }
    }
    
    // Render unified grid
    renderUnifiedGrid();

    // Update session stats
    updateSessionStats();

    // Update combined move history
    updateCombinedMoveHistory();
}

function renderUnifiedGrid() {
    if (!unifiedSessionData || !unifiedSessionData.sessions || unifiedSessionData.sessions.length === 0) return;

    const gridEl = document.getElementById('unifiedGrid');

    // Clear and rebuild grid
    gridEl.innerHTML = '';
    // Use the grid from the first session
    const firstSession = unifiedSessionData.sessions[0];
    const grid = firstSession.game_state ? firstSession.game_state.grid : null;

    if (!grid) return;

    // Pre-compute trail positions for all sessions
    const sessionTrails = new Map();
    const sessionRecentTrails = new Map();

    activeSessions.forEach((sessionInfo, sessionId) => {
        const trailPositions = new Set();
        const recentTrailPositions = new Set();

        if (sessionInfo.data && sessionInfo.data.game_state && sessionInfo.data.game_state.move_history) {
            const moveHistory = sessionInfo.data.game_state.move_history;
            const recentMoves = moveHistory.slice(-5); // Last 5 moves for each session

            moveHistory.forEach(move => {
                if (move.to_position && move.success) {
                    const key = `${move.to_position.x},${move.to_position.y}`;
                    trailPositions.add(key);
                }
            });

            recentMoves.forEach(move => {
                if (move.to_position && move.success) {
                    const key = `${move.to_position.x},${move.to_position.y}`;
                    recentTrailPositions.add(key);
                }
            });
        }

        sessionTrails.set(sessionId, trailPositions);
        sessionRecentTrails.set(sessionId, recentTrailPositions);
    });

    for (let y = 0; y < grid.length; y++) {
        const row = gridEl.insertRow();
        for (let x = 0; x < grid[y].length; x++) {
            const cell = row.insertCell();
            const cellData = grid[y][x];

            // Set cell class
            cell.className = `cell-${cellData.type}`;

            // Add cell icons (not cars - those are handled separately)
            const icon = cellIcons[cellData.type];
            if (icon && cellData.type !== 'road') {
                if (cellData.type === 'park') {
                    // Check which sessions visited this park
                    const visitedBy = [];
                    activeSessions.forEach((sessionInfo, sessionId) => {
                        if (cellData.id && sessionInfo.data && sessionInfo.data.game_state &&
                            sessionInfo.data.game_state.visited_parks &&
                            sessionInfo.data.game_state.visited_parks[cellData.id]) {
                            visitedBy.push(sessionInfo.color);
                        }
                    });

                    if (visitedBy.length > 0) {
                        // Show checkmark for visited parks
                        cell.innerHTML = `<span style="color: ${visitedBy[0]};">✓</span>`;
                        cell.classList.add('visited');
                    } else {
                        cell.textContent = icon;
                    }
                } else {
                    cell.textContent = icon;
                }
            }

            // Add trail dots for each session (after content, before cars)
            const posKey = `${x},${y}`;
            cell.style.position = 'relative';

            // Check if any session has a trail here
            activeSessions.forEach((sessionInfo, sessionId) => {
                const trailPositions = sessionTrails.get(sessionId);
                const recentTrailPositions = sessionRecentTrails.get(sessionId);

                // Skip if this is the current player position for this session
                const playerPos = sessionInfo.data && sessionInfo.data.game_state && sessionInfo.data.game_state.player_pos;
                const isCurrentPosition = playerPos && playerPos.x === x && playerPos.y === y;

                if (trailPositions && trailPositions.has(posKey) && !isCurrentPosition) {
                    const trailDot = document.createElement('div');
                    trailDot.className = 'trail-dot';
                    trailDot.style.backgroundColor = sessionInfo.color;

                    // Fade older trail dots
                    if (!recentTrailPositions.has(posKey)) {
                        trailDot.classList.add('fade');
                    }

                    cell.appendChild(trailDot);
                }
            });
        }
    }

    // Position any existing car elements
    activeSessions.forEach((sessionInfo, sessionId) => {
        if (sessionInfo.data && sessionInfo.data.game_state && sessionInfo.data.game_state.player_pos) {
            animateCarMovement(sessionId, sessionInfo.data.game_state.player_pos);
        }
    });
}

function updateSessionLegend() {
    const legendEl = document.getElementById('sessionLegend');
    legendEl.innerHTML = '';
    
    activeSessions.forEach((sessionInfo, sessionId) => {
        const legendItem = document.createElement('div');
        legendItem.style.cssText = 'display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: #f8f9fa; border-radius: 6px; border: 2px solid ' + sessionInfo.color;
        
        const status = sessionInfo.data.game_state?.game_over ?
            (sessionInfo.data.game_state?.victory ? '🏆' : '💔') : '🎮';
        
        // Create elements safely to prevent XSS
        const iconSpan = document.createElement('span');
        iconSpan.style.fontSize = '20px';
        iconSpan.textContent = sessionInfo.icon;

        const infoDiv = document.createElement('div');

        const sessionIdDiv = document.createElement('div');
        sessionIdDiv.style.fontWeight = '600';
        sessionIdDiv.style.color = '#171a20';
        sessionIdDiv.textContent = sessionId; // Safe text content

        const statsDiv = document.createElement('div');
        statsDiv.style.fontSize = '12px';
        statsDiv.style.color = '#666';
        statsDiv.textContent = `${status} Battery: ${sessionInfo.data.game_state?.battery || 0}/${sessionInfo.data.game_state?.max_battery || 0} • Parks: ${sessionInfo.data.game_state?.score || 0}`;

        infoDiv.appendChild(sessionIdDiv);
        infoDiv.appendChild(statsDiv);

        legendItem.appendChild(iconSpan);
        legendItem.appendChild(infoDiv);
        
        legendEl.appendChild(legendItem);
    });
}

function updateSessionStats() {
    const statsEl = document.getElementById('sessionStatsList');
    statsEl.innerHTML = '';
    
    activeSessions.forEach((sessionInfo, sessionId) => {
        const statItem = document.createElement('div');
        statItem.style.cssText = 'padding: 12px 16px; border-bottom: 1px solid #f4f4f4; position: relative;';
        
        const batteryPercent = ((sessionInfo.data.game_state?.battery || 0) / (sessionInfo.data.game_state?.max_battery || 1)) * 100;
        const progressPercent = ((sessionInfo.data.game_state?.score || 0) / (unifiedSessionData.total_parks || 1)) * 100;
        
        // Add flash animation for game over or victory
        let statusBadgeStyle = '';
        if (sessionInfo.data.game_state?.game_over) {
            if (sessionInfo.data.game_state?.victory) {
                statusBadgeStyle = 'background: #4caf50; color: white; animation: flashPulse 1s ease-in-out;';
            } else {
                statusBadgeStyle = 'background: #f44336; color: white; animation: flashPulse 1s ease-in-out;';
            }
        }
        
        // Create header div
        const headerDiv = document.createElement('div');
        headerDiv.style.cssText = 'display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;';

        const leftDiv = document.createElement('div');
        leftDiv.style.cssText = 'display: flex; align-items: center; gap: 8px;';

        const iconSpan = document.createElement('span');
        iconSpan.style.fontSize = '18px';
        iconSpan.textContent = sessionInfo.icon;

        const sessionSpan = document.createElement('span');
        sessionSpan.style.cssText = `font-weight: 600; color: ${sessionInfo.color};`;
        sessionSpan.textContent = sessionId; // Safe text content

        leftDiv.appendChild(iconSpan);
        leftDiv.appendChild(sessionSpan);

        const statusBadge = document.createElement('span');
        statusBadge.style.cssText = `font-size: 12px; padding: 2px 8px; border-radius: 4px; ${statusBadgeStyle}`;
        statusBadge.textContent = sessionInfo.data.game_state?.game_over ?
            (sessionInfo.data.game_state?.victory ? '🏆 Victory!' : '💔 Game Over') : '🎮 Active';

        headerDiv.appendChild(leftDiv);
        headerDiv.appendChild(statusBadge);

        // Battery progress
        const batteryDiv = document.createElement('div');
        batteryDiv.style.marginBottom = '4px';
        batteryDiv.innerHTML = `
            <div style="font-size: 11px; color: #666; margin-bottom: 2px;">Battery</div>
            <div style="height: 4px; background: #e0e0e0; border-radius: 2px; overflow: hidden;">
                <div style="height: 100%; width: ${batteryPercent}%; background: ${batteryPercent > 30 ? '#4caf50' : '#ff9800'}; transition: width 0.3s;"></div>
            </div>
        `;

        // Progress bar
        const progressDiv = document.createElement('div');
        progressDiv.style.marginBottom = '4px';
        progressDiv.innerHTML = `
            <div style="font-size: 11px; color: #666; margin-bottom: 2px;">Progress</div>
            <div style="height: 4px; background: #e0e0e0; border-radius: 2px; overflow: hidden;">
                <div style="height: 100%; width: ${progressPercent}%; background: ${sessionInfo.color}; transition: width 0.3s;"></div>
            </div>
        `;

        // Stats footer
        const statsDiv = document.createElement('div');
        statsDiv.style.cssText = 'display: flex; justify-content: space-between; align-items: center; margin-top: 6px;';

        const movesSpan = document.createElement('span');
        movesSpan.style.cssText = 'font-size: 11px; color: #666;';
        movesSpan.innerHTML = `Total Moves: <strong>${sessionInfo.data.game_state?.total_moves || 0}</strong>`;

        const parksSpan = document.createElement('span');
        parksSpan.style.cssText = 'font-size: 11px; color: #666;';
        parksSpan.textContent = `Parks: ${sessionInfo.data.game_state?.score || 0}/${unifiedSessionData.total_parks}`;

        statsDiv.appendChild(movesSpan);
        statsDiv.appendChild(parksSpan);

        // Append all to statItem
        statItem.appendChild(headerDiv);
        statItem.appendChild(batteryDiv);
        statItem.appendChild(progressDiv);
        statItem.appendChild(statsDiv);
        
        statsEl.appendChild(statItem);
    });
}

function closeGameOverlay() {
    const overlay = document.getElementById('game-over');
    overlay.classList.remove('active');
}

function resetGameAndCloseOverlay() {
    // Reset the game
    if (currentSessionId) {
        fetch(`/api/sessions/${currentSessionId}/reset`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            console.log('Game reset successfully');
            // Close the overlay
            closeGameOverlay();
        })
        .catch(error => {
            console.error('Failed to reset game:', error);
        });
    }
}

function resetCurrentGame() {
    if (currentSessionId) {
        fetch(`/api/sessions/${currentSessionId}/reset`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            console.log('Game reset successfully');
        })
        .catch(error => {
            console.error('Failed to reset game:', error);
        });
    }
}

// Make functions global so onclick can access them
window.copyPrompt = copyPrompt;
window.toggleMenu = toggleMenu;
window.toggleConfig = toggleConfig;
window.toggleCaveMode = toggleCaveMode;
window.closeGameOverlay = closeGameOverlay;
window.resetGameAndCloseOverlay = resetGameAndCloseOverlay;
window.resetCurrentGame = resetCurrentGame;
window.updateRadius = updateRadius;
window.setViewMode = setViewMode;
window.addSelectedSessionToUnified = addSelectedSessionToUnified;
window.refreshSessionList = refreshSessionList;
window.refreshSessionDropdown = refreshSessionDropdown;
window.joinSelectedSession = joinSelectedSession;
window.toggleMultiSelect = toggleMultiSelect;
window.joinMultipleSessions = joinMultipleSessions;

// Connect only after session is created/joined (removed automatic connection)

// Load cave mode settings on page load
loadCaveModeSettings();

// Load configurations on page load
loadConfigurations();

// Configuration preview data
let configsData = {};

// Session management functions
async function loadConfigPreview(configName) {
    try {
        // First get the list of configs if we don't have them cached
        if (Object.keys(configsData).length === 0) {
            const response = await fetch('/api/configs');
            const configList = await response.json();

            // Fetch full details for each config
            for (const config of configList) {
                try {
                    const name = config.filename.replace('.json', '');
                    const detailResponse = await fetch(`/api/configs/${name}`);
                    const details = await detailResponse.json();
                    configsData[name] = {
                        ...config,
                        ...details
                    };
                } catch (err) {
                    console.error(`Failed to load config ${config.filename}:`, err);
                }
            }
        }

        // Update preview with selected config
        updateConfigPreview(configName);
    } catch (error) {
        console.error('Failed to load config preview:', error);
        document.getElementById('previewStats').textContent = 'Failed to load preview';
    }
}

function updateConfigPreview(configName) {
    const config = configsData[configName];
    if (!config) {
        document.getElementById('previewStats').textContent = 'Loading...';
        return;
    }

    // Update stats
    const parkCount = config.layout.reduce((count, row) => 
        count + (row.match(/P/g) || []).length, 0);
    document.getElementById('previewStats').textContent = 
        `${config.grid_size}×${config.grid_size} • Battery: ${config.starting_battery}/${config.max_battery} • Parks: ${parkCount}`;

    // Update grid preview
    const previewGrid = document.getElementById('previewGrid');
    previewGrid.innerHTML = renderPreviewGrid(config.layout);
}

function renderPreviewGrid(layout) {
    let html = '<table>';
    layout.forEach(row => {
        html += '<tr>';
        for (let char of row) {
            let cellClass = '';
            let cellContent = '';
            switch(char) {
                case 'R': cellClass = 'preview-road'; break;
                case 'H': cellClass = 'preview-home'; cellContent = '🏠'; break;
                case 'P': cellClass = 'preview-park'; cellContent = '🌳'; break;
                case 'S': cellClass = 'preview-supercharger'; cellContent = '⚡'; break;
                case 'W': cellClass = 'preview-water'; cellContent = '💧'; break;
                case 'B': cellClass = 'preview-building'; cellContent = '🏢'; break;
            }
            html += `<td class="${cellClass}">${cellContent}</td>`;
        }
        html += '</tr>';
    });
    html += '</table>';
    return html;
}

// Load available sessions from the server
function loadAvailableSessions() {
    const loadingEl = document.getElementById('session-list-loading');
    const listEl = document.getElementById('session-list');
    const noSessionsEl = document.getElementById('no-sessions');
    const countEl = document.getElementById('session-count');
    
    // Show loading state
    loadingEl.style.display = 'flex';
    listEl.style.display = 'none';
    noSessionsEl.style.display = 'none';
    
    fetch('/api/sessions')
        .then(response => response.json())
        .then(data => {
            loadingEl.style.display = 'none';
            
            if (data.count === 0) {
                noSessionsEl.style.display = 'flex';
                countEl.textContent = 'No sessions';
            } else {
                listEl.style.display = 'block';
                countEl.textContent = `${data.count} session${data.count !== 1 ? 's' : ''}`;
                displaySessionList(data.sessions);
            }
        })
        .catch(error => {
            console.error('Failed to load sessions:', error);
            loadingEl.style.display = 'none';
            noSessionsEl.style.display = 'flex';
            countEl.textContent = 'Error loading';
        });
}

// Display the list of sessions
function displaySessionList(sessions) {
    const listEl = document.getElementById('session-list');
    listEl.innerHTML = '';

    sessions.forEach(session => {
        const sessionId = session.id || session.session_id;
        const itemEl = document.createElement('div');
        itemEl.className = 'session-item';
        if (multiSelectedSessions.has(sessionId)) {
            itemEl.classList.add('multi-selected');
        }

        // Determine status
        let statusClass = 'active';
        let statusText = 'Active';
        let statusIcon = '🎮';

        if (session.victory) {
            statusClass = 'victory';
            statusText = 'Victory';
            statusIcon = '🏆';
        } else if (session.game_over) {
            statusClass = 'gameover';
            statusText = 'Game Over';
            statusIcon = '💔';
        }

        // Calculate relative time
        const lastAccessed = new Date(session.last_accessed_at);
        const timeAgo = getRelativeTime(lastAccessed);

        itemEl.innerHTML = `
            <input type="checkbox" class="session-checkbox" id="check-${sessionId}"
                ${multiSelectedSessions.has(sessionId) ? 'checked' : ''}>
            <div class="session-item-content">
                <div class="session-item-header">
                    <span class="session-id">${sessionId}</span>
                    <span class="session-status ${statusClass}">${statusIcon} ${statusText}</span>
                </div>
                <div class="session-details">
                    <span class="session-detail">🗺️ ${session.config_name}</span>
                    <span class="session-detail">⚡ ${session.game_state?.battery || 0}/${session.game_state?.max_battery || 0}</span>
                    <span class="session-detail">🌳 ${session.game_state?.score || 0}</span>
                </div>
                <div class="session-time">Last active: ${timeAgo}</div>
            </div>
        `;

        // Add event listener to checkbox after creating the element
        const checkbox = itemEl.querySelector('.session-checkbox');
        checkbox.addEventListener('click', (event) => {
            event.stopPropagation();
            toggleMultiSelect(sessionId);
        });

        listEl.appendChild(itemEl);
    });

    // Update multi-select button state
    updateMultiSelectButton();
}

// Get relative time string
function getRelativeTime(date) {
    const now = new Date();
    const seconds = Math.floor((now - date) / 1000);
    
    if (seconds < 60) return 'just now';
    if (seconds < 120) return '1 minute ago';
    if (seconds < 3600) return `${Math.floor(seconds / 60)} minutes ago`;
    if (seconds < 7200) return '1 hour ago';
    if (seconds < 86400) return `${Math.floor(seconds / 3600)} hours ago`;
    if (seconds < 172800) return '1 day ago';
    return `${Math.floor(seconds / 86400)} days ago`;
}

// No longer needed - using checkbox selection only

// Toggle multi-select for a session
function toggleMultiSelect(sessionId) {
    if (multiSelectedSessions.has(sessionId)) {
        multiSelectedSessions.delete(sessionId);
    } else {
        multiSelectedSessions.add(sessionId);
    }

    // Update visual state
    const checkbox = document.getElementById(`check-${sessionId}`);
    if (checkbox) {
        checkbox.checked = multiSelectedSessions.has(sessionId);
    }

    // Update item visual state
    const items = document.querySelectorAll('.session-item');
    items.forEach(item => {
        const itemCheckbox = item.querySelector('.session-checkbox');
        if (itemCheckbox && itemCheckbox.id === `check-${sessionId}`) {
            if (multiSelectedSessions.has(sessionId)) {
                item.classList.add('multi-selected');
            } else {
                item.classList.remove('multi-selected');
            }
        }
    });

    updateMultiSelectButton();
}

// Update multi-select button state
function updateMultiSelectButton() {
    const button = document.getElementById('joinSessionBtn');
    const count = document.getElementById('selectedCount');
    const buttonText = document.getElementById('joinButtonText');

    if (button && count) {
        count.textContent = multiSelectedSessions.size;
        button.disabled = multiSelectedSessions.size === 0;

        if (multiSelectedSessions.size === 0) {
            button.style.background = '#ccc';
            button.style.cursor = 'not-allowed';
            if (buttonText) buttonText.textContent = 'Selected';
        } else if (multiSelectedSessions.size === 1) {
            button.style.background = '#4caf50';
            button.style.cursor = 'pointer';
            if (buttonText) buttonText.textContent = 'Session';
        } else {
            button.style.background = '#9c27b0';
            button.style.cursor = 'pointer';
            if (buttonText) buttonText.textContent = 'Sessions';
        }
    }
}

// Join multiple selected sessions (internal function)
function joinMultipleSessions() {
    const sessionIds = Array.from(multiSelectedSessions);
    console.log(`Joining multiple sessions: ${sessionIds.join(', ')}`);

    // Hide session screen and switch to multi-session view
    document.getElementById('sessionScreen').style.display = 'none';
    document.getElementById('viewModeSelector').style.display = 'block';
    viewMode = 'multi';
    setViewMode('multi');

    // Load the selected sessions in multi-session view
    loadUnifiedSessions(sessionIds);
}

// Refresh the session list
function refreshSessionList() {
    loadAvailableSessions();
}

// Refresh the session dropdown in multi-session view
function refreshSessionDropdown() {
    const button = event.target;
    const originalText = button.innerHTML;

    // Show loading state
    button.innerHTML = '⟳';
    button.style.animation = 'spin 1s linear infinite';
    button.disabled = true;

    // Call the existing function to reload the dropdown
    loadSessionDropdown();

    // Reset button state after a short delay
    setTimeout(() => {
        button.innerHTML = originalText;
        button.style.animation = 'none';
        button.disabled = false;
    }, 1000);
}

// Join the selected session(s)
function joinSelectedSession() {
    // Check if we have checkbox selections
    if (multiSelectedSessions.size === 0) {
        showSessionError('Please select at least one session using the checkboxes');
        return;
    }

    if (multiSelectedSessions.size === 1) {
        // Single session - join normally
        const sessionId = Array.from(multiSelectedSessions)[0];
        joinSession(sessionId);
    } else {
        // Multiple sessions - switch to multi-session view
        joinMultipleSessions();
    }
}

function joinSession(sessionId) {
    console.log(`🚀 Join Session button clicked! Session ID: ${sessionId || 'none'}`);

    if (!sessionId) {
        showSessionError('Please select a session to join');
        return;
    }
    
    // Verify session exists
    fetch(`/api/sessions/${sessionId}`)
        .then(response => {
            if (!response.ok) {
                throw new Error('Session not found');
            }
            return response.json();
        })
        .then(sessionData => {
            // Session exists, connect to it
            currentSessionId = sessionId;
            hideSessionScreen();
            showGameDashboard();
            updateCurrentSessionInfo(sessionId, sessionData.game_config?.name || 'Unknown Config');
            connect();
            fetchInitialState();
        })
        .catch(error => {
            console.error('Failed to join session:', error);
            showSessionError('Session not found. Please check the session ID and try again.');
        });
}

function createNewGameSession() {
    const selectedConfig = document.getElementById('config-selector').value;
    
    const requestBody = { config_name: selectedConfig };
    
    fetch('/api/sessions', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(requestBody)
    })
    .then(response => response.json())
    .then(data => {
        if (data.id) {
            currentSessionId = data.id;
            hideSessionScreen();
            showGameDashboard();
            updateCurrentSessionInfo(data.id, data.config_name || selectedConfig);
            connect();
            fetchInitialState();
        } else {
            showSessionError('Failed to create session. Please try again.');
        }
    })
    .catch(error => {
        console.error('Failed to create session:', error);
        showSessionError('Error creating session. Please try again.');
    });
}

function showSessionScreen() {
    document.getElementById('sessionScreen').style.display = 'flex';
    document.getElementById('gameDashboard').style.display = 'none';
    
    // Close WebSocket connection
    if (ws) {
        ws.close();
    }
}

function hideSessionScreen() {
    document.getElementById('sessionScreen').style.display = 'none';
    // Show view mode selector when game starts
    document.getElementById('viewModeSelector').style.display = 'block';
}

function showGameDashboard() {
    document.getElementById('gameDashboard').style.display = 'flex';
}

// Switch to multi-session view
function switchToMultiSessionView() {
    // Hide single session dashboard
    document.getElementById('gameDashboard').style.display = 'none';

    // Show multi-session dashboard
    document.getElementById('multiSessionDashboard').style.display = 'flex';

    // Update view mode
    viewMode = 'multi';

    // Start multi-session WebSocket connections
    startMultiSessionWebSockets();

    console.log('Switched to multi-session view');
}

function updateCurrentSessionInfo(sessionId, configName) {
    document.getElementById('currentSessionId').textContent = sessionId;
    document.getElementById('currentConfigName').textContent = configName;
}

function showSessionError(message) {
    const errorDiv = document.getElementById('sessionError');
    errorDiv.textContent = message;
    errorDiv.style.display = 'block';
    
    // Hide error after 5 seconds
    setTimeout(() => {
        errorDiv.style.display = 'none';
    }, 5000);
}

function fetchInitialState() {
    if (!currentSessionId) {
        console.error('No session ID available');
        return;
    }
    fetch(`/api/sessions/${currentSessionId}/state`)
        .then(response => response.json())
        .then(data => {
            queueAnimation(data);
        })
        .catch(error => console.error('Failed to fetch initial state:', error));
}

// Load available configurations from server
function loadAvailableConfigurations() {
    fetch('/api/configs')
        .then(response => response.json())
        .then(configs => {
            const selector = document.getElementById('config-selector');
            selector.innerHTML = ''; // Clear existing options
            
            configs.forEach(config => {
                const option = document.createElement('option');
                const configKey = config.filename.replace('.json', '');
                option.value = configKey;
                option.textContent = `${config.name} - ${config.description}`;
                selector.appendChild(option);
            });
            
            // Select first config by default and load its preview
            if (configs.length > 0) {
                const firstConfig = configs[0].filename.replace('.json', '');
                selector.value = firstConfig;
                loadConfigPreview(firstConfig);
            }
        })
        .catch(error => {
            console.error('Failed to load configurations:', error);
            const selector = document.getElementById('config-selector');
            selector.innerHTML = '<option value="">Failed to load configurations</option>';
        });
}

// Make session functions global
window.joinSession = joinSession;
window.createNewGameSession = createNewGameSession;
window.showSessionScreen = showSessionScreen;

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', function() {
    // Initialize config preview
    document.getElementById('config-selector').addEventListener('change', function() {
        loadConfigPreview(this.value);
    });

    // Load available configurations on startup
    loadAvailableConfigurations();

    // Load available sessions on startup
    loadAvailableSessions();

    // Auto-refresh session list every 5 seconds when on session screen
    sessionListRefreshTimer = setInterval(() => {
        const sessionScreen = document.getElementById('sessionScreen');
        if (sessionScreen && sessionScreen.style.display !== 'none') {
            loadAvailableSessions();
        }
    }, 5000);
});

// Clean up on page unload
window.addEventListener('beforeunload', function() {
    // Clear all timers
    if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }

    if (sessionListRefreshTimer) {
        clearInterval(sessionListRefreshTimer);
        sessionListRefreshTimer = null;
    }

    // Clear pending reconnections
    pendingReconnections.forEach(timer => clearTimeout(timer));
    pendingReconnections.clear();

    // Close all WebSockets
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
    }

    sessionWebSockets.forEach(ws => {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.close();
        }
    });
});

    // Keyboard controls for single session mode
    document.addEventListener('keydown', function(e) {
        // Only work in single session mode and when game dashboard is visible
        if (viewMode !== 'single' || document.getElementById('gameDashboard').style.display === 'none') {
            return;
        }

        let action = null;
        switch(e.key) {
            case 'ArrowUp':
            case 'w':
            case 'W':
                action = 'up';
                break;
            case 'ArrowDown':
            case 's':
            case 'S':
                action = 'down';
                break;
            case 'ArrowLeft':
            case 'a':
            case 'A':
                action = 'left';
                break;
            case 'ArrowRight':
            case 'd':
            case 'D':
                action = 'right';
                break;
            case 'r':
            case 'R':
                action = 'reset';
                break;
        }

        if (action && currentSessionId) {
            e.preventDefault();

            // Handle reset separately
            if (action === 'reset') {
        fetch(`/api/sessions/${currentSessionId}/reset`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            console.log('Game reset successfully');
        })
        .catch(error => {
            console.error('Reset failed:', error);
        });
            } else {
                // Handle movement
        fetch(`/api/sessions/${currentSessionId}/move`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ direction: action })
        })
        .then(response => response.json())
        .then(data => {
            console.log('Move completed:', action, data.player_pos);
        })
        .catch(error => {
            console.error('Move failed:', error);
        });
            }
        }
    });

    // Check for sessionId query parameter and auto-join if present
    const urlParams = new URLSearchParams(window.location.search);
    const sessionIdParam = urlParams.get('sessionId');

    if (sessionIdParam) {
        // Check if multiple session IDs are provided (comma-separated)
        const sessionIds = sessionIdParam.split(',').map(id => id.trim()).filter(id => id);

        if (sessionIds.length > 1) {
            // Multiple sessions - switch to multi-session mode
            console.log(`Auto-joining multiple sessions from URL parameter: ${sessionIds.join(', ')}`);
            viewMode = 'multi';
            document.getElementById('sessionScreen').style.display = 'none';
            document.getElementById('gameDashboard').style.display = 'flex';
            switchToMultiSessionView();

            // Load and join each session
            loadUnifiedSessions(sessionIds);
        } else {
            // Single session - join normally
            console.log(`Auto-joining session from URL parameter: ${sessionIds[0]}`);
            joinSession(sessionIds[0]);
        }
    } else {
        // Initialize the app - show session screen by default
        document.getElementById('sessionScreen').style.display = 'flex';
        document.getElementById('gameDashboard').style.display = 'none';
    }
