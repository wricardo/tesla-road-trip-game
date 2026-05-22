<script lang="ts">
	import { getContextClient, gql } from '@urql/svelte';
	import { onMount } from 'svelte';

	type CellType = 'R' | 'H' | 'P' | 'S' | 'W' | 'B';

	interface GridCell {
		type: CellType;
	}

	interface Config {
		mapId?: string;
		name: string;
		description: string;
	}

	interface MapConfig {
		name: string;
		description: string;
		gridSize: number;
		maxBattery: number;
		startingBattery: number;
		layout: string[];
		wallCrashEndsGame: boolean;
	}

	const client = getContextClient();

	const cellIcons: Record<CellType, string> = {
		R: '⬜',
		H: '🏠',
		P: '🌳',
		S: '⚡',
		W: '💧',
		B: '🏢'
	};

	const cellLabels: Record<CellType, string> = {
		R: 'Road',
		H: 'Home',
		P: 'Park',
		S: 'Supercharger',
		W: 'Water',
		B: 'Building'
	};

	let gridSize = $state(10);
	let gridData = $state<CellType[][]>([]);
	let currentTool = $state<CellType>('R');
	let configs = $state<Config[]>([]);

	// Form fields
	let mapName = $state('');
	let description = $state('');
	let maxBattery = $state(20);
	let startingBattery = $state(20);
	let wallCrashEndsGame = $state(true);

	// Stats
	let parkCount = $state(0);
	let homeCount = $state(0);
	let superchargerCount = $state(0);
	let obstacleCount = $state(0);

	// UI state
	let validationMessage = $state('');
	let validationType = $state<'error' | 'warning' | 'success' | ''>('');
	let isSaving = $state(false);
	let selectedConfigId = $state('');

	onMount(() => {
		initializeGrid(gridSize);
		loadAvailableConfigs();
	});

	function initializeGrid(size: number) {
		gridSize = size;
		gridData = Array(size)
			.fill(null)
			.map(() => Array(size).fill('R' as CellType));
		updateStats();
	}

	function paintCell(row: number, col: number) {
		gridData[row][col] = currentTool;
		gridData = gridData; // Trigger reactivity
		updateStats();
	}

	function fillGrid(type: CellType) {
		gridData = Array(gridSize)
			.fill(null)
			.map(() => Array(gridSize).fill(type));
		updateStats();
	}

	function clearGrid() {
		if (confirm('Clear all cells to roads?')) {
			fillGrid('R');
		}
	}

	function resizeGrid() {
		const newSize = parseInt((document.getElementById('gridSize') as HTMLSelectElement)?.value || '10');
		if (confirm(`Resize grid to ${newSize}x${newSize}? This will clear current data.`)) {
			initializeGrid(newSize);
		}
	}

	function updateStats() {
		parkCount = 0;
		homeCount = 0;
		superchargerCount = 0;
		obstacleCount = 0;

		for (let row = 0; row < gridSize; row++) {
			for (let col = 0; col < gridSize; col++) {
				const cell = gridData[row][col];
				if (cell === 'P') parkCount++;
				else if (cell === 'H') homeCount++;
				else if (cell === 'S') superchargerCount++;
				else if (cell === 'W' || cell === 'B') obstacleCount++;
			}
		}
	}

	function selectCellType(type: CellType) {
		currentTool = type;
	}

	function validateMap(): boolean {
		const errors: string[] = [];

		if (parkCount === 0) errors.push('At least 1 park required');
		if (homeCount === 0) errors.push('At least 1 home required');
		if (!mapName.trim()) errors.push('Map name is required');

		if (errors.length > 0) {
			validationType = 'error';
			validationMessage = '❌ Validation Failed:\n' + errors.join('\n');
			return false;
		}

		validationType = 'success';
		validationMessage = '✅ Map is valid and ready to save!';
		return true;
	}

	async function saveConfiguration() {
		if (!validateMap()) return;

		isSaving = true;

		const layout = gridData.map((row) => row.join(''));
		const config: MapConfig = {
			name: mapName,
			description,
			gridSize,
			maxBattery,
			startingBattery,
			layout,
			wallCrashEndsGame,
		};

		try {
			const CREATE_MAP = gql`
				mutation CreateMap($name: String!, $map: GameMapInput!) {
					createMap(name: $name, map: $map) {
						name
					}
				}
			`;

			const result = await client.mutation(CREATE_MAP, { name: mapName, map: config }).toPromise();

			if (result.error) throw result.error;

			validationType = 'success';
			validationMessage = '✅ Map saved successfully! Redirecting...';

			setTimeout(() => {
				window.location.href = '/';
			}, 2000);
		} catch (error) {
			validationType = 'error';
			validationMessage = `❌ Error: ${error instanceof Error ? error.message : 'Unknown error'}`;
		} finally {
			isSaving = false;
		}
	}

	async function loadAvailableConfigs() {
		try {
			const MAPS_QUERY = gql`
				query {
					maps {
						mapId
						name
						description
					}
				}
			`;

			const result = await client.query(MAPS_QUERY, {}).toPromise();
			if (result.data?.maps) {
				configs = result.data.maps;
			}
		} catch (error) {
			console.error('Failed to load maps:', error);
		}
	}

	async function loadSelectedConfig() {
		if (!selectedConfigId) {
			mapName = '';
			description = '';
			maxBattery = 20;
			startingBattery = 20;
			wallCrashEndsGame = true;
			initializeGrid(10);
			validationMessage = '';
			return;
		}

		try {
			const MAP_QUERY = gql`
				query GetMap($name: String!) {
					map(name: $name) {
						name
						description
						gridSize
						maxBattery
						startingBattery
						layout
						wallCrashEndsGame
					}
				}
			`;

			const result = await client.query(MAP_QUERY, { name: selectedConfigId }).toPromise();
			if (result.error) throw result.error;

			const map = result.data?.map;
			mapName = map.name;
			description = map.description;
			maxBattery = map.maxBattery;
			startingBattery = map.startingBattery;
			wallCrashEndsGame = map.wallCrashEndsGame;
			gridSize = map.gridSize;

			gridData = map.layout.map((row: string) => row.split('') as CellType[]);
			updateStats();

			validationType = 'success';
			validationMessage = '✅ Map loaded for editing';
		} catch (error) {
			validationType = 'error';
			validationMessage = `❌ Error loading map: ${error instanceof Error ? error.message : 'Unknown error'}`;
		}
	}
</script>

<svelte:head>
	<title>Map Editor — Tesla Road Trip</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 flex flex-col">
	<!-- Header -->
	<div class="bg-gradient-to-r from-[#667eea] to-[#764ba2] text-white px-6 py-8">
		<div class="max-w-7xl mx-auto">
			<h1 class="text-3xl font-light">🗺️ Map Editor</h1>
			<p class="text-sm text-blue-100 mt-2">Create and edit game maps • Click cells to paint • Validate and save</p>
		</div>
	</div>

	<!-- Main Content -->
	<div class="flex-1 max-w-7xl mx-auto w-full px-4 py-8 grid grid-cols-1 lg:grid-cols-[1fr_400px] gap-6">
		<!-- Grid Editor Section -->
		<div class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6 flex flex-col">
			<div class="overflow-x-auto mb-6">
				<table class="border-collapse">
					<tbody>
						{#each gridData as row, rowIdx}
							<tr>
								{#each row as cell, colIdx}
									<td
										onclick={() => paintCell(rowIdx, colIdx)}
										class="w-8 h-8 border border-gray-200 text-center text-sm cursor-pointer hover:bg-gray-100 transition-colors"
										role="button"
										tabindex="0"
										onkeydown={(e) => e.key === 'Enter' && paintCell(rowIdx, colIdx)}
									>
										{cellIcons[cell]}
									</td>
								{/each}
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Grid Controls -->
			<div class="space-y-3 pt-4 border-t border-gray-200">
				<div>
					<label class="text-sm font-medium text-gray-700 mb-2 block">Load Map:</label>
					<select
						bind:value={selectedConfigId}
						onchange={loadSelectedConfig}
						class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
					>
						<option value="">-- New Map --</option>
						{#each configs as config}
							<option value={config.mapId}>{config.name}</option>
						{/each}
					</select>
				</div>

				<div>
					<label for="gridSize" class="text-sm font-medium text-gray-700 mb-2 block">Grid Size:</label>
					<select
						id="gridSize"
						bind:value={gridSize}
						onchange={resizeGrid}
						class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
					>
						<option value="5">5×5</option>
						<option value="8">8×8</option>
						<option value="10">10×10</option>
						<option value="15">15×15</option>
						<option value="20">20×20</option>
						<option value="25">25×25</option>
						<option value="30">30×30</option>
					</select>
				</div>

				<div class="grid grid-cols-3 gap-2">
					<button
						onclick={() => fillGrid('R')}
						class="px-3 py-2 bg-gray-100 hover:bg-gray-200 text-sm font-medium rounded-lg transition-colors"
					>
						Fill Roads
					</button>
					<button
						onclick={() => fillGrid('B')}
						class="px-3 py-2 bg-gray-100 hover:bg-gray-200 text-sm font-medium rounded-lg transition-colors"
					>
						Fill Buildings
					</button>
					<button
						onclick={clearGrid}
						class="px-3 py-2 bg-red-100 hover:bg-red-200 text-red-600 text-sm font-medium rounded-lg transition-colors"
					>
						Clear All
					</button>
				</div>
			</div>

			<!-- Validation Message -->
			{#if validationMessage}
				<div
					class={`mt-4 p-3 rounded-lg text-sm ${
						validationType === 'error'
							? 'bg-red-50 border border-red-200 text-red-700'
							: validationType === 'warning'
							  ? 'bg-yellow-50 border border-yellow-200 text-yellow-700'
							  : 'bg-green-50 border border-green-200 text-green-700'
					}`}
				>
					{validationMessage}
				</div>
			{/if}
		</div>

		<!-- Settings Panel -->
		<div class="space-y-6">
			<!-- Cell Palette -->
			<div class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
				<h3 class="font-medium text-gray-900 mb-4">Cell Types</h3>
				<div class="grid grid-cols-2 gap-2">
					{#each Object.entries(cellLabels) as [type, label]}
						<button
							onclick={() => selectCellType(type as CellType)}
							class={`p-3 rounded-lg text-sm font-medium transition-all border-2 ${
								currentTool === type
									? 'border-blue-500 bg-blue-50 text-blue-700'
									: 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50'
							}`}
						>
							<div class="text-lg">{cellIcons[type as CellType]}</div>
							<div class="text-xs mt-1">{label}</div>
						</button>
					{/each}
				</div>
				<p class="text-xs text-gray-500 mt-3">Current: <strong>{cellLabels[currentTool]}</strong></p>
			</div>

			<!-- Statistics -->
			<div class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
				<h3 class="font-medium text-gray-900 mb-4">Statistics</h3>
				<div class="grid grid-cols-2 gap-3">
					<div class="p-3 bg-gray-50 rounded-lg">
						<div class="text-xs text-gray-500 mb-1">Parks</div>
						<div class="text-2xl font-bold text-gray-900">{parkCount}</div>
					</div>
					<div class="p-3 bg-gray-50 rounded-lg">
						<div class="text-xs text-gray-500 mb-1">Homes</div>
						<div class="text-2xl font-bold text-gray-900">{homeCount}</div>
					</div>
					<div class="p-3 bg-gray-50 rounded-lg">
						<div class="text-xs text-gray-500 mb-1">Chargers</div>
						<div class="text-2xl font-bold text-gray-900">{superchargerCount}</div>
					</div>
					<div class="p-3 bg-gray-50 rounded-lg">
						<div class="text-xs text-gray-500 mb-1">Obstacles</div>
						<div class="text-2xl font-bold text-gray-900">{obstacleCount}</div>
					</div>
				</div>
			</div>

			<!-- Configuration -->
			<div class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
				<h3 class="font-medium text-gray-900 mb-4">Configuration</h3>
				<div class="space-y-4">
					<div>
						<label for="mapName" class="block text-sm font-medium text-gray-700 mb-1">Map Name *</label>
						<input
							id="mapName"
							bind:value={mapName}
							type="text"
							placeholder="e.g., custom_maze"
							class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
						/>
						<p class="text-xs text-gray-500 mt-1">Lowercase, use underscores</p>
					</div>

					<div>
						<label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
						<input
							id="description"
							bind:value={description}
							type="text"
							placeholder="e.g., A challenging maze layout"
							class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
						/>
					</div>

					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="maxBattery" class="block text-sm font-medium text-gray-700 mb-1">Max Battery</label>
							<input
								id="maxBattery"
								bind:value={maxBattery}
								type="number"
								min="10"
								max="100"
								class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							/>
						</div>

						<div>
							<label for="startingBattery" class="block text-sm font-medium text-gray-700 mb-1">Starting Battery</label>
							<input
								id="startingBattery"
								bind:value={startingBattery}
								type="number"
								min="10"
								max="100"
								class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							/>
						</div>
					</div>

					<div class="flex items-center gap-2">
						<input
							id="wallCrash"
							bind:checked={wallCrashEndsGame}
							type="checkbox"
							class="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-2 focus:ring-blue-500"
						/>
						<label for="wallCrash" class="text-sm text-gray-700">Wall collision ends game</label>
					</div>

					<button
						onclick={saveConfiguration}
						disabled={isSaving}
						class="w-full px-4 py-2 mt-4 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-medium rounded-lg transition-colors"
					>
						{isSaving ? '💾 Saving...' : '💾 Save Map'}
					</button>
				</div>
			</div>
		</div>
	</div>
</div>
