// Tesla Road Trip Game - Main JavaScript
let ws = null;
let gameState = null;
let caveMode = false;
let caveRadius = 2;
let previousVisibleCells = new Set();
let currentSessionId = '';

// Hybrid session support
let activeSessionId = '';        // The session user controls
let observerSessionIds = [];     // Other sessions to watch (same config)
let hybridMode = false;          // True when session loaded
let unifiedSessionData = null;
let sessionWebSockets = new Map(); // sessionId -> WebSocket
let sessionColors = ['#e31d23', '#2196f3', '#4caf50', '#ff9800', '#9c27b0', '#00bcd4'];
let sessionIcons = ['🚗', '🚙', '🏎️', '🚘', '🚖', '🚓'];
let activeSessions = new Map(); // sessionId -> {color, icon, data}
let pendingReconnections = new Map(); // Track pending reconnection timers
let carElements = new Map(); // sessionId -> DOM element

// Animation queue for smooth state transitions
let stateQueue = [];
let isAnimating = false;
let processingState = false; // Prevent race condition
const ANIMATION_DELAY_MS = 500; // Half second between moves

const cellIcons = {
    'home': '🏠',
    'park': '🌳',
    'supercharger': '⚡',
    'water': '💧',
    'building': '🏢',
    'road': '',
    'player': '🚗'
};

// === Animation Queue System ===

function queueStateUpdate(newState) {
    // Deep clone to preserve each state snapshot
    const stateCopy = JSON.parse(JSON.stringify(newState));
    stateQueue.push(stateCopy);

    // Start processing if not already animating
    if (!isAnimating) {
        processNextState();
    }
}

function processNextState() {
    if (processingState || stateQueue.length === 0) {
        if (stateQueue.length === 0) {
            isAnimating = false;
        }
        return;
    }

    processingState = true;
    isAnimating = true;
    const nextState = stateQueue.shift();

    // Update game state
    gameState = nextState;

    // Render
    if (gameState && gameState.grid) {
        renderUnifiedGridHybrid();
        applyCaveMode();
        updateActiveSessionStats();
    }

    // Schedule next state
    setTimeout(() => {
        processingState = false;
        processNextState();
    }, ANIMATION_DELAY_MS);
}

function renderImmediately() {
    // For initial load and direct updates, bypass queue
    if (gameState && gameState.grid) {
        renderUnifiedGridHybrid();
        applyCaveMode();
        updateActiveSessionStats();
    }
}

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

    // Always use unifiedGrid
    const gridEl = document.getElementById('unifiedGrid');
    if (!gridEl || !gridEl.rows) return;

    const numRows = gameState.grid.length;
    const numCols = gameState.grid[0]?.length || 0;

    for (let y = 0; y < numRows; y++) {
        const row = gridEl.rows[y];
        if (!row) continue;

        for (let x = 0; x < numCols; x++) {
            const cell = row.cells[x];
            if (!cell) continue;

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
        }
    }

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
        const toggleEl = document.getElementById('cave-mode-toggle');
        if (toggleEl) toggleEl.checked = caveMode;
    }

    if (savedRadius !== null) {
        caveRadius = parseInt(savedRadius);
        const radiusEl = document.getElementById('cave-radius');
        const valueEl = document.getElementById('radius-value');
        if (radiusEl) radiusEl.value = caveRadius;
        if (valueEl) valueEl.textContent = caveRadius;
    }

    console.log('Cave Mode settings loaded - enabled:', caveMode, 'radius:', caveRadius);
}

// Copy prompt function
function copyPrompt(event) {
    const promptEl = document.getElementById('ai-prompt-content');
    const preEl = promptEl?.querySelector('pre');
    const userTaskEl = document.getElementById('user-task');

    if (!promptEl || !preEl || !userTaskEl) {
        console.error('Required elements not found for copyPrompt');
        return;
    }

    const systemPrompt = preEl.textContent;
    const userTask = userTaskEl.value.trim();

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

function closeGameOverlay() {
    const overlay = document.getElementById('game-over');
    if (overlay) overlay.classList.remove('active');
}

function resetGameAndCloseOverlay() {
    if (activeSessionId) {
        fetch(`/api/sessions/${activeSessionId}/reset`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        })
        .then(response => response.json())
        .then(data => {
            console.log('Game reset successfully');
            closeGameOverlay();
        })
        .catch(error => {
            console.error('Failed to reset game:', error);
        });
    }
}

function resetCurrentGame() {
    const sessionId = activeSessionId || currentSessionId;
    if (sessionId) {
        fetch(`/api/sessions/${sessionId}/reset`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
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
window.startGame = startGame;

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
function startGame() {
    const selectedConfig = document.getElementById('config-selector').value;
    const resumeId = document.getElementById('resume-session-id').value.trim();

    if (resumeId) {
        // Resume existing session
        loadHybridSession(resumeId).then(() => {
            document.getElementById('sessionScreen').style.display = 'none';
            document.getElementById('unifiedDashboard').style.display = 'flex';
        }).catch(() => showSessionError('Session not found: ' + resumeId));
    } else {
        // Create new session
        fetch('/api/sessions', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ config_name: selectedConfig })
        })
        .then(r => r.json())
        .then(data => {
            if (data.id) {
                loadHybridSession(data.id).then(() => {
                    document.getElementById('sessionScreen').style.display = 'none';
                    document.getElementById('unifiedDashboard').style.display = 'flex';
                });
            } else {
                showSessionError('Failed to create session');
            }
        })
        .catch(e => showSessionError('Error: ' + e.message));
    }
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
window.showSessionScreen = showSessionScreen;

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', function() {
    // Initialize config preview
    document.getElementById('config-selector').addEventListener('change', function() {
        loadConfigPreview(this.value);
    });

    // Load available configurations on startup
    loadAvailableConfigurations();

    // Initialize hybrid keyboard controls
    setupHybridKeyboardControls();

    // Auto-refresh observer sessions every 10 seconds while dashboard is visible
    setInterval(() => {
        const dashboard = document.getElementById('unifiedDashboard');
        if (dashboard && dashboard.style.display !== 'none' && hybridMode) {
            refreshObserverSessions();
        }
    }, 10000);
});

// Clean up on page unload
window.addEventListener('beforeunload', function() {
    // Clear pending reconnections
    pendingReconnections.forEach(timer => clearTimeout(timer));
    pendingReconnections.clear();

    // Close all WebSockets
    closeAllWebSockets();
});

    // Check for sessionId query parameter and auto-join if present
    const urlParams = new URLSearchParams(window.location.search);
    const sessionIdParam = urlParams.get('sessionId');

    if (sessionIdParam) {
        // Use first session ID (ignore comma-separated for now, always use hybrid)
        const sessionIds = sessionIdParam.split(',').map(id => id.trim()).filter(id => id);
        const primarySessionId = sessionIds[0];

        console.log(`Auto-joining session from URL parameter: ${primarySessionId}`);
        loadHybridSession(primarySessionId).then(() => {
            document.getElementById('sessionScreen').style.display = 'none';
            document.getElementById('unifiedDashboard').style.display = 'flex';
        }).catch(error => {
            console.error('Failed to auto-join session:', error);
            // Fall back to session screen
            document.getElementById('sessionScreen').style.display = 'flex';
        });
    } else {
        // Initialize the app - show session screen by default
        document.getElementById('sessionScreen').style.display = 'flex';
    }

// ===== HYBRID VIEW MODE IMPLEMENTATION =====

function loadHybridSession(sessionId) {
    activeSessionId = sessionId;
    hybridMode = true;

    // Load active session details
    return fetch(`/api/sessions/${sessionId}`)
        .then(response => response.json())
        .then(sessionData => {
            gameState = sessionData.game_state;
            currentSessionId = sessionId; // Backward compatibility

            // Auto-discover other sessions with same config
            const configName = sessionData.config_name;
            unifiedSessionData = sessionData;

            if (configName) {
                loadObserverSessions(configName);
            }

            // Setup WebSockets
            setupHybridWebSockets();

            // Initialize visualization
            renderUnifiedGridHybrid();
            applyCaveMode(); // Apply if enabled
            updateActiveSessionStats();
        })
        .catch(error => {
            console.error('Failed to load hybrid session:', error);
            showSessionError('Failed to load session: ' + error.message);
        });
}

function loadObserverSessions(configName) {
    fetch(`/api/sessions/unified?configName=${encodeURIComponent(configName)}`)
        .then(response => response.json())
        .then(data => {
            // Filter out active session
            observerSessionIds = data.sessions
                .map(s => s.session_id)
                .filter(id => id !== activeSessionId);

            console.log(`Found ${observerSessionIds.length} other players`);

            // Connect to observer WebSockets
            observerSessionIds.forEach(id => connectObserverSession(id));

            // Update observer panel
            updateObserverSessionsList();
        })
        .catch(error => {
            console.log('No unified sessions endpoint available, skipping observer discovery');
            observerSessionIds = [];
            updateObserverSessionsList();
        });
}

// === WebSocket Management ===

function setupHybridWebSockets() {
    closeAllWebSockets(); // Close existing
    connectActiveSession(activeSessionId);
    observerSessionIds.forEach(id => connectObserverSession(id));
}

function connectActiveSession(sessionId) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws?session=${sessionId}`);

    ws.onopen = () => {
        console.log(`🎮 Active session ${sessionId} connected`);
        updateConnectionStatus('connected');
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        // WebSocket might send game_state nested or directly
        const newState = data.game_state || data;

        // Queue state update for smooth animation
        if (newState && newState.grid) {
            queueStateUpdate(newState);
        }
    };

    ws.onclose = () => {
        console.log('Active session disconnected');
        updateConnectionStatus('disconnected');
        // Reconnect after 2 seconds
        setTimeout(() => {
            if (hybridMode && activeSessionId) {
                connectActiveSession(activeSessionId);
            }
        }, 2000);
    };
}

function connectObserverSession(sessionId) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsObserver = new WebSocket(`${protocol}//${window.location.host}/ws?session=${sessionId}`);

    wsObserver.onopen = () => {
        console.log(`👁️ Observer ${sessionId} connected`);
    };

    wsObserver.onmessage = (event) => {
        const data = JSON.parse(event.data);
        // Normalize data structure: always use game_state property
        const gameState = data.game_state || data;

        // Update observer data in activeSessions immediately
        if (!activeSessions.has(sessionId)) {
            const color = sessionColors[activeSessions.size % sessionColors.length];
            const icon = sessionIcons[activeSessions.size % sessionIcons.length];
            activeSessions.set(sessionId, { color, icon, data: {} });
        }
        activeSessions.get(sessionId).data = { game_state: gameState };

        // Update observer list
        updateObserverSessionsList();

        // Trigger re-render if not animating active session
        // Observer moves are shown in real-time
        if (!isAnimating) {
            renderImmediately();
        }
        // If animating, the active session's queue will show observers during its renders
    };

    wsObserver.onclose = () => {
        console.log(`Observer ${sessionId} disconnected`);
        // Try reconnect after 3 seconds
        setTimeout(() => {
            if (hybridMode && observerSessionIds.includes(sessionId)) {
                connectObserverSession(sessionId);
            }
        }, 3000);
    };

    sessionWebSockets.set(sessionId, wsObserver);
}

function closeAllWebSockets() {
    if (ws) ws.close();
    sessionWebSockets.forEach(wsConn => wsConn.close());
    sessionWebSockets.clear();
}

// === UI Update Functions ===

function updateActiveSessionStats() {
    if (!gameState) return;

    document.getElementById('activeSessionId').textContent = activeSessionId;
    document.getElementById('activeBattery').textContent =
        `${gameState.battery}/${gameState.max_battery}`;

    const batteryPercent = (gameState.battery / gameState.max_battery) * 100;
    const batteryBar = document.getElementById('activeBatteryBar');
    batteryBar.style.width = batteryPercent + '%';

    // Battery bar color
    if (batteryPercent > 50) {
        batteryBar.style.background = '#4caf50';
    } else if (batteryPercent > 25) {
        batteryBar.style.background = '#ff9800';
    } else {
        batteryBar.style.background = '#f44336';
    }

    const totalParks = countTotalParks(gameState.grid);
    document.getElementById('activeScore').textContent =
        `${gameState.score}/${totalParks}`;

    document.getElementById('activeMoves').textContent =
        gameState.total_moves || 0;

    document.getElementById('activeMessage').textContent =
        gameState.message || 'Ready';

    // Config name
    if (unifiedSessionData && unifiedSessionData.config_name) {
        document.getElementById('unifiedConfigName').textContent =
            unifiedSessionData.config_name;
    }
}

function updateObserverSessionsList() {
    const listEl = document.getElementById('observerSessionsList');
    if (!listEl) return;

    listEl.innerHTML = '';

    if (observerSessionIds.length === 0) {
        listEl.innerHTML = '<p class="no-observers">No other players</p>';
        return;
    }

    observerSessionIds.forEach(sessionId => {
        const sessionInfo = activeSessions.get(sessionId);
        if (!sessionInfo || !sessionInfo.data) return;

        const state = sessionInfo.data.game_state;
        if (!state) return;

        const itemEl = document.createElement('div');
        itemEl.className = 'observer-item';
        itemEl.innerHTML = `
            <div class="observer-icon" style="color: ${sessionInfo.color};">
                ${sessionInfo.icon}
            </div>
            <div class="observer-info">
                <div class="observer-id">${sessionId}</div>
                <div class="observer-stats">
                    ⚡${state.battery}/${state.max_battery} •
                    🌳${state.score} •
                    📍${state.total_moves} moves
                </div>
            </div>
        `;
        listEl.appendChild(itemEl);
    });
}

function refreshObserverSessions() {
    if (!gameState || !unifiedSessionData) return;
    const configName = unifiedSessionData.config_name;
    if (configName) {
        loadObserverSessions(configName);
    }
}

function showSessionScreen() {
    // Disconnect all WebSockets
    closeAllWebSockets();

    // Clear animation queue
    stateQueue = [];
    isAnimating = false;
    processingState = false;

    // Reset state
    hybridMode = false;
    activeSessionId = '';
    observerSessionIds = [];

    // Hide dashboard, show session screen
    document.getElementById('unifiedDashboard').style.display = 'none';
    document.getElementById('sessionScreen').style.display = 'flex';
}

function countTotalParks(grid) {
    let count = 0;
    grid.forEach(row => {
        row.forEach(cell => {
            if (cell.type === 'park') count++;
        });
    });
    return count;
}

// === Grid Rendering (Hybrid - Active + Observer Distinction) ===

function renderUnifiedGridHybrid() {
    if (!gameState) return;

    const grid = gameState.grid;
    const gridEl = document.getElementById('unifiedGrid');
    gridEl.innerHTML = '';

    // Build grid structure
    for (let y = 0; y < grid.length; y++) {
        const row = gridEl.insertRow();
        for (let x = 0; x < grid[y].length; x++) {
            const cell = row.insertCell();
            const cellData = grid[y][x];

            cell.className = `cell-${cellData.type}`;

            // Add icon for special cells
            if (cellIcons[cellData.type]) {
                cell.textContent = cellIcons[cellData.type];
            }

            // Mark visited parks
            if (cellData.type === 'park' && cellData.visited) {
                cell.classList.add('visited');
            }
        }
    }

    // Clear old trail dots
    document.querySelectorAll('.trail-dot').forEach(el => el.remove());

    // Render active player trail first
    if (gameState && gameState.trail) {
        gameState.trail.forEach(pos => {
            const cell = gridEl.rows[pos.y]?.cells[pos.x];
            if (cell) {
                const dot = document.createElement('div');
                dot.className = 'trail-dot';
                dot.style.background = '#e31d23'; // Red for active player
                dot.style.opacity = '1';
                cell.appendChild(dot);
            }
        });
    }

    // Render trails for observer sessions
    activeSessions.forEach((sessionInfo, sessionId) => {
        const state = sessionInfo.data?.game_state;
        if (!state || !state.trail) return;

        state.trail.forEach(pos => {
            const cell = gridEl.rows[pos.y]?.cells[pos.x];
            if (cell) {
                const dot = document.createElement('div');
                dot.className = 'trail-dot';
                dot.style.background = sessionInfo.color;
                cell.appendChild(dot);
            }
        });
    });

    // Clear old car elements
    carElements.forEach(el => el.remove());
    carElements.clear();

    // Render active player car (YOUR car)
    if (gameState && gameState.player_pos) {
        const pos = gameState.player_pos;
        const cell = gridEl.rows[pos.y]?.cells[pos.x];
        if (cell) {
            const car = document.createElement('div');
            car.className = 'multi-car active-player-car';
            car.textContent = gameState.game_over ? '💥' : '🚗';
            car.style.fontSize = '48px';
            car.style.filter = 'drop-shadow(0 0 12px rgba(231,29,35,0.8))';
            car.style.zIndex = '20';
            cell.appendChild(car);
            carElements.set('active', car);
        }
    }

    // Render observer cars (OTHER players)
    activeSessions.forEach((sessionInfo, sessionId) => {
        if (sessionId === activeSessionId) return; // Skip active (already rendered)

        const state = sessionInfo.data?.game_state;
        if (!state || !state.player_pos) return;

        const pos = state.player_pos;
        const cell = gridEl.rows[pos.y]?.cells[pos.x];
        if (!cell) return;

        const car = document.createElement('div');
        car.className = 'multi-car observer-car';
        car.textContent = state.game_over ? '💥' : (sessionInfo.icon || '🚙');
        car.style.color = sessionInfo.color || '#2196f3';
        car.style.fontSize = '36px';
        car.style.opacity = '0.7';
        car.style.zIndex = '10';
        car.title = `Session ${sessionId}`;

        cell.appendChild(car);
        carElements.set(sessionId, car);
    });
}

// === Keyboard Controls for Active Session Only ===

function setupHybridKeyboardControls() {
    document.addEventListener('keydown', function(e) {
        if (!hybridMode || !activeSessionId) {
            return;
        }

        // Prevent if session screen visible
        if (document.getElementById('sessionScreen').style.display !== 'none') {
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

        if (action) {
            e.preventDefault();
            if (action === 'reset') {
                // Call dedicated reset endpoint
                fetch(`/api/sessions/${activeSessionId}/reset`, {
                    method: 'POST'
                })
                .then(response => response.json())
                .then(result => {
                    // WebSocket will handle state update
                });
            } else {
                // Send move to active session only
                fetch(`/api/sessions/${activeSessionId}/move`, {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({direction: action})
                })
                .then(response => response.json())
                .then(result => {
                    if (!result.success && result.message) {
                        console.warn('Move failed:', result.message);
                    }
                    // WebSocket will handle state update
                });
            }
        }
    });
}

// === Cave Mode Sync Functions for Unified Dashboard ===

function updateCaveRadius(newRadius) {
    caveRadius = parseInt(newRadius);

    // Update both displays
    const display1 = document.getElementById('caveRadiusDisplay');
    const display2 = document.getElementById('radius-value');
    if (display1) display1.textContent = caveRadius;
    if (display2) display2.textContent = caveRadius;

    // Sync sliders
    const slider1 = document.getElementById('caveRadiusSlider');
    const slider2 = document.getElementById('cave-radius');
    if (slider1) slider1.value = caveRadius;
    if (slider2) slider2.value = caveRadius;

    // Save to localStorage
    localStorage.setItem('caveModeRadius', caveRadius);

    // Apply immediately if cave mode is enabled
    if (caveMode && gameState) {
        applyCaveMode();
    }
}

// Unified toggleCaveMode that syncs both checkboxes
const originalToggleCaveMode = toggleCaveMode;
toggleCaveMode = function() {
    const checkbox1 = document.getElementById('caveModeToggle');
    const checkbox2 = document.getElementById('cave-mode-toggle');

    // Determine which was clicked and sync the other
    if (checkbox1 && checkbox2) {
        const newState = checkbox1.checked || checkbox2.checked;
        checkbox1.checked = newState;
        checkbox2.checked = newState;
        caveMode = newState;
    } else if (checkbox1) {
        caveMode = checkbox1.checked;
    } else if (checkbox2) {
        caveMode = checkbox2.checked;
    }

    // Save to localStorage
    localStorage.setItem('caveModeEnabled', caveMode);

    // Apply immediately if game state exists
    if (gameState) {
        applyCaveMode();
    }

    console.log('Cave Mode toggled:', caveMode);
};

// === Helper Functions ===

function updateConnectionStatus(status) {
    const statusEl = document.getElementById('status');
    if (!statusEl) return;

    if (status === 'connected') {
        statusEl.textContent = 'Connected';
        statusEl.className = 'connection-status connected';
    } else {
        statusEl.textContent = 'Disconnected';
        statusEl.className = 'connection-status disconnected';
    }
}
