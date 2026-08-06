<script lang="ts">
	import { getContextClient, gql } from '@urql/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { directionGlyph, standardDirectionalCellConfigs, directionalChars } from '$lib/directional';
	import uiAuthConfig from '$lib/config/ui-auth.json';

	type CellType = string;

	interface Config {
		mapId: string;
		name: string;
		description: string;
	}

	type LegendEntry = { key: string; value: string };
	type CellConfigEntry = { key: string; type: string; allowedDirections: string[] };

	interface MapConfig {
		name: string;
		description: string;
		gridSize: number;
		maxBattery: number;
		startingBattery: number;
		layout: string[];
		legend: LegendEntry[];
		cellConfigs: CellConfigEntry[];
		wallCrashEndsGame: boolean;
	}

	const client = getContextClient();
	const mapQueryPassword = uiAuthConfig.uiMapPassword ?? '';

	const cellLabels: Record<string, string> = {
		R: 'Road',
		H: 'Home',
		P: 'Park',
		S: 'Supercharger',
		W: 'Water',
		B: 'Building',
		'|': 'North/South road',
		'-': 'East/West road',
		'^': 'North-only road',
		v: 'South-only road',
		'>': 'East-only road',
		'<': 'West-only road',
		J: 'North/East turn',
		L: 'North/West turn',
		'7': 'South/East turn',
		r: 'South/West turn'
	};

	const cellClasses: Record<string, string> = {
		R: 'bg-white',
		H: 'bg-red-500 ring-1 ring-red-200',
		P: 'bg-emerald-500',
		S: 'bg-yellow-400',
		W: 'bg-blue-400',
		B: 'bg-slate-700',
		'|': 'bg-white text-orange-500 font-bold',
		'-': 'bg-white text-orange-500 font-bold',
		'^': 'bg-white text-orange-500 font-bold',
		v: 'bg-white text-orange-500 font-bold',
		'>': 'bg-white text-orange-500 font-bold',
		'<': 'bg-white text-orange-500 font-bold',
		J: 'bg-white text-orange-500 font-bold',
		L: 'bg-white text-orange-500 font-bold',
		'7': 'bg-white text-orange-500 font-bold',
		r: 'bg-white text-orange-500 font-bold'
	};

	const defaultLegend: LegendEntry[] = [
		{ key: 'R', value: 'road' },
		{ key: 'H', value: 'home' },
		{ key: 'P', value: 'park' },
		{ key: 'S', value: 'supercharger' },
		{ key: 'W', value: 'water' },
		{ key: 'B', value: 'building' }
	];

	let gridSize = $state(10);
	let gridData = $state<CellType[][]>([]);
	let currentTool = $state<CellType>('R');
	let currentCellConfigs = $state<CellConfigEntry[]>(standardDirectionalCellConfigs);
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
	let isValidating = $state(false);
	let isLoadingMap = $state(false);
	let selectedConfigId = $state('');
	let originalMapId = $state('');
	let duplicateSourceId = $state('');
	let isDuplicateMode = $state(false);
	let lastSavedMapId = $state('');
	let hasSavedMap = $state(false);
	const isEditMode = $derived(Boolean(originalMapId));

	onMount(async () => {
		initializeGrid(gridSize);
		await loadAvailableConfigs();
		const requestedMap = $page.url.searchParams.get('map');
		const requestedDuplicate = ['1', 'true', 'yes'].includes(($page.url.searchParams.get('duplicate') ?? '').toLowerCase());
		if (requestedMap) {
			if (configs.some((config) => config.mapId === requestedMap)) {
				selectedConfigId = requestedMap;
				await loadSelectedConfig({ duplicate: requestedDuplicate });
			} else {
				validationType = 'error';
				validationMessage = `Could not find map "${requestedMap}". Starting a new map instead.`;
			}
		}
	});

	function suggestDuplicateName(sourceMapID: string) {
		const base = `${sourceMapID}_copy`;
		if (!mapExists(base)) return base;
		let suffix = 2;
		while (mapExists(`${base}_${suffix}`)) suffix++;
		return `${base}_${suffix}`;
	}

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

	function displayLabel(cell: CellType) {
		return cellLabels[cell] ?? 'Directional road';
	}

	function displayClass(cell: CellType) {
		return cellClasses[cell] ?? 'bg-white text-orange-500 font-bold';
	}

	function displayGlyph(cell: CellType) {
		return directionalChars.has(cell) ? directionGlyph(currentCellConfigs.find((entry) => entry.key === cell)?.allowedDirections ?? []) : '';
	}

	function resetEditor() {
		selectedConfigId = '';
		originalMapId = '';
		duplicateSourceId = '';
		isDuplicateMode = false;
		lastSavedMapId = '';
		hasSavedMap = false;
		mapName = '';
		description = '';
		maxBattery = 20;
		startingBattery = 20;
		wallCrashEndsGame = true;
		currentTool = 'R';
		currentCellConfigs = standardDirectionalCellConfigs;
		initializeGrid(10);
		validationMessage = '';
		validationType = '';
		goto('/editor', { replaceState: true, noScroll: true });
	}

	function mapExists(mapId: string) {
		return configs.some((config) => config.mapId === mapId);
	}

	function saveAsNameChanged() {
		return mapName.trim() !== '' && mapName.trim() !== originalMapId;
	}

	function validateMap(options: { saveAsNew?: boolean } = {}): boolean {
		const errors: string[] = [];

		if (parkCount === 0) errors.push('At least 1 park required');
		if (homeCount === 0) errors.push('At least 1 home required');
		if (!mapName.trim()) errors.push('Map name is required');
		if (options.saveAsNew && isEditMode && !saveAsNameChanged()) {
			errors.push('Change the map name before using Save as new');
		}
		if (options.saveAsNew && mapName.trim() !== originalMapId && mapExists(mapName.trim())) {
			errors.push('A map with this name already exists');
		}

		if (errors.length > 0) {
			validationType = 'error';
			validationMessage = '❌ Validation Failed:\n' + errors.join('\n');
			return false;
		}

		validationType = 'success';
		validationMessage = '✅ Map is valid and ready to save!';
		return true;
	}

	function buildMapConfig(): MapConfig {
		const layout = gridData.map((row) => row.join(''));
		return {
			name: mapName,
			description,
			gridSize,
			maxBattery,
			startingBattery,
			layout,
			legend: defaultLegend,
			cellConfigs: currentCellConfigs,
			wallCrashEndsGame
		};
	}

	async function persistMap(targetMapId: string, successMessage: string) {
		isSaving = true;

		try {
			const CREATE_MAP = gql`
				mutation CreateMap($name: String!, $map: GameMapInput!) {
					createMap(name: $name, map: $map) {
						name
					}
				}
			`;

			const result = await client.mutation(CREATE_MAP, { name: targetMapId, map: buildMapConfig() }).toPromise();

			if (result.error) throw result.error;

			await loadAvailableConfigs();
			selectedConfigId = targetMapId;
			originalMapId = targetMapId;
			lastSavedMapId = targetMapId;
			hasSavedMap = true;
			validationType = 'success';
			validationMessage = successMessage;
			goto(`/editor?map=${encodeURIComponent(targetMapId)}`, { replaceState: true, noScroll: true });
		} catch (error) {
			validationType = 'error';
			validationMessage = `❌ Error: ${error instanceof Error ? error.message : 'Unknown error'}`;
		} finally {
			isSaving = false;
		}
	}

	async function saveConfiguration() {
		if (!validateMap()) return;
		const targetMapId = isEditMode ? originalMapId : mapName.trim();
		await persistMap(targetMapId, isEditMode ? 'Map updated successfully.' : 'Map saved successfully.');
	}

	async function saveAsNew() {
		if (!validateMap({ saveAsNew: true })) return;
		await persistMap(mapName.trim(), 'New map saved successfully.');
	}

	async function validateWithSolver() {
		if (!validateMap()) return;
		isValidating = true;
		try {
			const VALIDATE_MAP = gql`
				mutation ValidateMap($map: GameMapInput!) {
					validateMap(map: $map) {
						valid
						winnable
						message
						error
					}
				}
			`;

			const result = await client.mutation(VALIDATE_MAP, { map: buildMapConfig() }).toPromise();
			if (result.error) throw result.error;

			const validation = result.data?.validateMap;
			if (!validation) throw new Error('No validation result returned');

			validationType = validation.valid ? 'success' : 'error';
			validationMessage = validation.valid
				? `✅ Solver validation passed: ${validation.message}`
				: `❌ Solver validation failed: ${validation.error ?? validation.message}`;
		} catch (error) {
			validationType = 'error';
			validationMessage = `❌ Error: ${error instanceof Error ? error.message : 'Unknown error'}`;
		} finally {
			isValidating = false;
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

			const result = await client.query(MAPS_QUERY, {}, { requestPolicy: 'network-only' }).toPromise();
			if (result.data?.maps) {
				configs = result.data.maps;
			}
		} catch (error) {
			console.error('Failed to load maps:', error);
		}
	}

	async function loadSelectedConfig(options: { duplicate?: boolean } = {}) {
		if (!selectedConfigId) {
			resetEditor();
			return;
		}

		isLoadingMap = true;
		hasSavedMap = false;
		try {
			const MAP_QUERY = gql`
				query GetMap($name: String!, $password: String) {
					map(name: $name, password: $password) {
						name
						description
						gridSize
						maxBattery
						startingBattery
						layout
						cellConfigs { key type allowedDirections }
						wallCrashEndsGame
					}
				}
			`;

			const result = await client.query(MAP_QUERY, { name: selectedConfigId, password: mapQueryPassword || null }, { requestPolicy: 'network-only' }).toPromise();
			if (result.error) throw result.error;

			const map = result.data?.map;
			if (!map) throw new Error('Map not found');
			mapName = map.name;
			description = map.description;
			maxBattery = map.maxBattery;
			startingBattery = map.startingBattery;
			wallCrashEndsGame = map.wallCrashEndsGame;
			gridSize = map.gridSize;

			currentCellConfigs = map.cellConfigs?.length ? map.cellConfigs : standardDirectionalCellConfigs;
			gridData = map.layout.map((row: string) => row.split('') as CellType[]);
			if (options.duplicate) {
				originalMapId = '';
				duplicateSourceId = selectedConfigId;
				isDuplicateMode = true;
				mapName = suggestDuplicateName(selectedConfigId);
				lastSavedMapId = '';
			} else {
				originalMapId = selectedConfigId;
				duplicateSourceId = '';
				isDuplicateMode = false;
				lastSavedMapId = selectedConfigId;
			}
			updateStats();

			validationType = 'success';
			validationMessage = options.duplicate
				? `Loaded ${selectedConfigId} as a duplicate draft. Save to create a new map.`
				: `Loaded ${map.name} for editing.`;
			goto(
				options.duplicate
					? `/editor?map=${encodeURIComponent(selectedConfigId)}&duplicate=1`
					: `/editor?map=${encodeURIComponent(selectedConfigId)}`,
				{ replaceState: true, noScroll: true }
			);
		} catch (error) {
			originalMapId = '';
			duplicateSourceId = '';
			isDuplicateMode = false;
			validationType = 'error';
			validationMessage = `Error loading map: ${error instanceof Error ? error.message : 'Unknown error'}`;
		} finally {
			isLoadingMap = false;
		}
	}
</script>

<svelte:head>
	<title>Map Editor — Tesla Road Trip</title>
</svelte:head>

<div class="bg-white border-b border-[#e8e8e8]">
	<div class="max-w-7xl mx-auto px-6 py-4">
		<div class="flex items-center justify-between gap-4">
			<div>
				<p class="text-xs font-bold uppercase tracking-widest text-red-500 mb-1">🗺️ Map editor</p>
				<h1 class="text-xl font-light text-[#171a20] tracking-tight">
					{isEditMode ? `Edit ${mapName || originalMapId}` : isDuplicateMode ? `Duplicate ${duplicateSourceId}` : 'Design a road trip map'}
				</h1>
			</div>
			<div class="flex flex-wrap gap-3">
				<button type="button" onclick={resetEditor} class="border border-gray-200 bg-white text-[#393c41] text-sm px-4 py-2 rounded-full hover:border-gray-400 transition-colors">New map</button>
				<a href="/maps" class="bg-[#393c41] text-white text-sm px-4 py-2 rounded-full hover:bg-black transition-colors">View maps</a>
			</div>
		</div>
	</div>
</div>

<div class="max-w-7xl mx-auto px-6 py-10">
	<div class="grid grid-cols-1 lg:grid-cols-[1fr_400px] gap-6 items-start">
		<section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
			<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
				<div>
					<h2 class="text-xl font-light text-[#393c41]">Grid layout</h2>
					<p class="text-sm text-gray-400 mt-0.5">{isLoadingMap ? 'Loading selected map…' : 'Click or press Enter on cells to paint with the selected terrain.'}</p>
				</div>
				<div class="flex flex-wrap gap-2">
					{#if isEditMode}
						<span class="shrink-0 text-[11px] text-gray-500 bg-[#f7f7f7] rounded-full px-2 py-1">Editing {originalMapId}</span>
					{:else if isDuplicateMode}
						<span class="shrink-0 text-[11px] text-gray-500 bg-[#f7f7f7] rounded-full px-2 py-1">Duplicating {duplicateSourceId}</span>
					{/if}
					<span class="shrink-0 text-[11px] text-gray-400 bg-[#f7f7f7] rounded-full px-2 py-1">{gridSize}×{gridSize}</span>
				</div>
			</div>

			{#if isLoadingMap}
				<div class="rounded-2xl bg-[#f7f7f7] border border-gray-100 p-8 mb-6 text-center text-sm text-gray-400">Loading map…</div>
			{:else}
			<div class="overflow-x-auto rounded-2xl bg-[#f7f7f7] p-3 border border-gray-100 mb-6">
				<div class="grid gap-0.5 min-w-fit" style={`grid-template-columns: repeat(${gridSize}, 2rem);`} aria-label="Editable map grid">
					{#each gridData as row, rowIdx}
						{#each row as cell, colIdx}
							<button
								type="button"
								onclick={() => paintCell(rowIdx, colIdx)}
								class={`w-8 h-8 rounded-[3px] border border-white/70 text-xs cursor-pointer hover:ring-2 hover:ring-gray-300 focus:outline-none focus:ring-2 focus:ring-[#393c41] transition-all flex items-center justify-center ${displayClass(cell)}`}
								aria-label={`${displayLabel(cell)} at row ${rowIdx + 1}, column ${colIdx + 1}`}
								title={displayLabel(cell)}
							>
								{displayGlyph(cell)}
								<span class="sr-only">{displayLabel(cell)}</span>
							</button>
						{/each}
					{/each}
				</div>
			</div>
			{/if}

			<div class="flex items-center gap-4 pt-4 border-t border-gray-100">
				<div class="w-40">
					<label for="gridSize" class="text-xs font-semibold text-[#393c41] mb-1.5 block">Grid size</label>
					<select
						id="gridSize"
						bind:value={gridSize}
						onchange={resizeGrid}
						class="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm bg-white focus:outline-none focus:border-gray-400"
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
				<div class="flex-1 flex justify-end">
					<button type="button" onclick={clearGrid} class="border border-red-200 bg-red-50 text-red-500 text-sm px-4 py-2.5 rounded-full hover:bg-red-100 transition-colors">Clear all</button>
				</div>
			</div>

			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4 pt-4 border-t border-gray-100">
				<div class="p-3 bg-[#f7f7f7] rounded-xl"><div class="text-xs text-gray-400 mb-1">Parks</div><div class="text-2xl font-light text-[#393c41]">{parkCount}</div></div>
				<div class="p-3 bg-[#f7f7f7] rounded-xl"><div class="text-xs text-gray-400 mb-1">Homes</div><div class="text-2xl font-light text-[#393c41]">{homeCount}</div></div>
				<div class="p-3 bg-[#f7f7f7] rounded-xl"><div class="text-xs text-gray-400 mb-1">Chargers</div><div class="text-2xl font-light text-[#393c41]">{superchargerCount}</div></div>
				<div class="p-3 bg-[#f7f7f7] rounded-xl"><div class="text-xs text-gray-400 mb-1">Obstacles</div><div class="text-2xl font-light text-[#393c41]">{obstacleCount}</div></div>
			</div>

			{#if validationMessage}
				<div
					class={`mt-4 px-5 py-4 rounded-2xl text-sm whitespace-pre-line ${
						validationType === 'error'
							? 'bg-red-50 border border-red-100 text-red-600'
							: validationType === 'warning'
							  ? 'bg-yellow-50 border border-yellow-100 text-yellow-700'
							  : 'bg-emerald-50 border border-emerald-100 text-emerald-700'
					}`}
				>
					{validationMessage}
					{#if hasSavedMap && validationType === 'success'}
						<div class="mt-3 flex flex-wrap gap-2 whitespace-normal">
							<a href={`/maps`} class="border border-emerald-200 bg-white/70 text-emerald-700 text-xs px-3 py-1.5 rounded-full hover:bg-white transition-colors">View maps</a>
							<a href={`/?map=${lastSavedMapId}`} class="border border-emerald-200 bg-white/70 text-emerald-700 text-xs px-3 py-1.5 rounded-full hover:bg-white transition-colors">Create session</a>
						</div>
					{/if}
				</div>
			{/if}
		</section>

		<aside class="space-y-6">
			<section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
				<h2 class="text-xl font-light text-[#393c41] mb-1">Cell palette</h2>
				<p class="text-sm text-gray-400 mb-4">Selected: <span class="font-medium text-[#393c41]">{cellLabels[currentTool]}</span></p>
				<div class="grid grid-cols-2 gap-2">
					{#each Object.entries(cellLabels) as [type, label]}
						<button
							type="button"
							onclick={() => selectCellType(type as CellType)}
							class={`p-3 rounded-2xl text-sm font-medium transition-all border ${
								currentTool === type
									? 'border-[#393c41] bg-[#f7f7f7] text-[#171a20] shadow-sm'
									: 'border-gray-200 bg-white text-gray-500 hover:border-gray-400 hover:text-[#393c41]'
							}`}
						>
							<span class={`flex items-center justify-center h-7 w-7 rounded-md mx-auto mb-2 border border-white/70 ${displayClass(type as CellType)}`}>{displayGlyph(type as CellType)}</span>
							<span class="text-xs">{label}</span>
						</button>
					{/each}
				</div>
			</section>

			<section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
				<h2 class="text-xl font-light text-[#393c41] mb-1">{isEditMode ? 'Edit configuration' : 'Configuration'}</h2>
				<p class="text-sm text-gray-400 mb-5">{isEditMode ? `Saving changes updates ${originalMapId}. Change the name and use Save as new to duplicate.` : isDuplicateMode ? `This is a duplicate draft of ${duplicateSourceId}. Save map will create a new map.` : 'Matches the GameMap schema used by saved maps.'}</p>
				<div class="space-y-4">
					<div>
						<label for="mapName" class="block text-xs font-semibold text-[#393c41] mb-1.5">Map name *</label>
						<input id="mapName" bind:value={mapName} type="text" placeholder="e.g., custom_maze" class="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm bg-white focus:outline-none focus:border-gray-400" />
						<p class="text-xs text-gray-400 mt-1">Use a stable lowercase identifier with underscores.</p>
					</div>

					<div>
						<label for="description" class="block text-xs font-semibold text-[#393c41] mb-1.5">Description</label>
						<input id="description" bind:value={description} type="text" placeholder="e.g., A challenging maze layout" class="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm bg-white focus:outline-none focus:border-gray-400" />
					</div>

					<div class="grid grid-cols-2 gap-3">
						<div>
							<label for="maxBattery" class="block text-xs font-semibold text-[#393c41] mb-1.5">Max battery</label>
							<input id="maxBattery" bind:value={maxBattery} type="number" min="10" max="100" class="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm bg-white focus:outline-none focus:border-gray-400" />
						</div>

						<div>
							<label for="startingBattery" class="block text-xs font-semibold text-[#393c41] mb-1.5">Starting battery</label>
							<input id="startingBattery" bind:value={startingBattery} type="number" min="10" max="100" class="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm bg-white focus:outline-none focus:border-gray-400" />
						</div>
					</div>

					<label for="wallCrash" class="flex items-center gap-2 text-sm text-gray-500">
						<input id="wallCrash" bind:checked={wallCrashEndsGame} type="checkbox" class="w-4 h-4 rounded border-gray-300 text-[#393c41] focus:ring-[#393c41]" />
						Wall collision ends game
					</label>

					<button onclick={saveConfiguration} disabled={isSaving || isLoadingMap || isValidating} class="w-full bg-[#393c41] text-white text-sm px-4 py-3 rounded-full hover:bg-black transition-colors disabled:opacity-50 mt-2">
						{isSaving ? 'Saving…' : isEditMode ? 'Save changes' : 'Save map'}
					</button>
					<button type="button" onclick={validateWithSolver} disabled={isSaving || isLoadingMap || isValidating} class="w-full border border-gray-200 bg-white text-[#393c41] text-sm px-4 py-3 rounded-full hover:border-gray-400 transition-colors disabled:opacity-50">
						{isValidating ? 'Validating…' : 'Validate solvable'}
					</button>
					{#if isEditMode}
						<button type="button" onclick={saveAsNew} disabled={isSaving || isLoadingMap || isValidating} class="w-full border border-gray-200 bg-white text-[#393c41] text-sm px-4 py-3 rounded-full hover:border-gray-400 transition-colors disabled:opacity-50">
							Save as new
						</button>
					{/if}
				</div>
			</section>
		</aside>
	</div>
</div>
