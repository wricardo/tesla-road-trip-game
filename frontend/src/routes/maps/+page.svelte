<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { directionsForChar, directionGlyph, terrainGlyph, type CellConfigEntry } from '$lib/directional';
	import { MAPS_QUERY, MAP_QUERY } from '$lib/queries';
	import uiAuthConfig from '$lib/config/ui-auth.json';

	type MapInfo = {
		mapId: string;
		name: string;
		description: string;
		gridSize: number;
		maxBattery: number;
	};
	type LegendEntry = { key: string; value: string };
	type GameMap = {
		name: string;
		description: string;
		gridSize: number;
		maxBattery: number;
		startingBattery: number;
		layout: string[];
		legend: LegendEntry[];
		cellConfigs: CellConfigEntry[];
	};
	type MapCard = MapInfo & { preview?: GameMap; error?: string };

	const client = getContextClient();
	const mapQueryPassword = uiAuthConfig.uiMapPassword ?? '';
	const mapsResult = queryStore({ client, query: gql(MAPS_QUERY) });

	let cards = $state<MapCard[]>([]);
	let loadingPreviews = $state(false);
	let requestID = 0;

	$effect(() => {
		const maps: MapInfo[] = $mapsResult?.data?.maps ?? [];
		if (!maps.length) {
			cards = [];
			return;
		}

		const current = ++requestID;
		loadingPreviews = true;
		cards = maps.map((map) => ({ ...map }));

		Promise.all(
			maps.map(async (map) => {
				const result = await client.query(gql(MAP_QUERY), { name: map.mapId, password: mapQueryPassword || null }).toPromise();
				if (result.error) return { ...map, error: result.error.message };
				return { ...map, preview: result.data?.map as GameMap };
			})
		).then((loaded) => {
			if (current !== requestID) return;
			cards = loaded;
			loadingPreviews = false;
		});
	});

	function tileType(char: string, legend: LegendEntry[] = [], cellConfigs: CellConfigEntry[] = []) {
		return cellConfigs.find((entry) => entry.key === char)?.type ?? legend.find((entry) => entry.key === char)?.value ?? 'road';
	}

	function tileClass(char: string, legend: LegendEntry[] = [], cellConfigs: CellConfigEntry[] = []) {
		switch (tileType(char, legend, cellConfigs)) {
			case 'home': return 'bg-red-500 ring-1 ring-red-200';
			case 'park': return 'bg-emerald-500';
			case 'supercharger': return 'bg-yellow-400';
			case 'water': return 'bg-blue-400';
			case 'building': return 'bg-slate-700';
			default: return 'bg-white';
		}
	}

	function countType(map: MapCard, type: string): number {
		const p = map.preview;
		if (!p) return 0;
		let n = 0;
		for (const row of p.layout) for (const ch of row) if (tileType(ch, p.legend, p.cellConfigs) === type) n++;
		return n;
	}

	let search = $state('');
	let sortBy = $state<'name' | 'size'>('name');
	const visible = $derived.by(() => {
		const q = search.trim().toLowerCase();
		return cards
			.filter((m) => !q || `${m.name} ${m.mapId} ${m.description}`.toLowerCase().includes(q))
			.sort((a, b) => (sortBy === 'size' ? a.gridSize - b.gridSize || a.name.localeCompare(b.name) : a.name.localeCompare(b.name)));
	});
</script>

<svelte:head>
	<title>Maps — Tesla Road Trip</title>
</svelte:head>

<div class="bg-white border-b border-[#e8e8e8]">
	<div class="max-w-7xl mx-auto px-6 py-12 lg:py-16">
		<p class="text-xs font-bold uppercase tracking-widest text-red-600 mb-4">🗺️ Map gallery</p>
		<div class="flex flex-col lg:flex-row lg:items-end lg:justify-between gap-6">
			<div>
				<h1 class="text-4xl lg:text-5xl font-light text-[#171a20] tracking-tight mb-4">Choose your road trip</h1>
				<p class="text-lg text-gray-500 font-light max-w-2xl">
					Preview every built-in map before creating a session. Compare routes, chargers, parks, and obstacles at a glance.
				</p>
			</div>
			<div class="flex flex-wrap gap-3">
				<a href="/" class="bg-[#393c41] text-white text-sm px-5 py-3 rounded-full hover:bg-black transition-colors">+ Create session</a>
				<a href="/editor" class="border border-gray-200 bg-white text-[#393c41] text-sm px-5 py-3 rounded-full hover:border-gray-400 transition-colors">Design a map →</a>
			</div>
		</div>
	</div>
</div>

<div class="max-w-7xl mx-auto px-6 py-10">
	<div class="flex flex-wrap items-end justify-between gap-4 mb-4">
		<div>
			<h2 class="text-xl font-light text-[#393c41]">All maps</h2>
			<p class="text-sm text-gray-500 mt-0.5">
				{visible.length}{visible.length === cards.length ? '' : ` of ${cards.length}`} maps{#if loadingPreviews} · loading previews…{/if}
			</p>
		</div>
		<div class="flex flex-wrap items-center gap-3">
			<input
				type="search"
				bind:value={search}
				placeholder="Search maps"
				aria-label="Search maps"
				class="min-w-[12rem] border border-gray-200 rounded-full px-4 py-2 text-sm bg-white"
			/>
			<label class="flex items-center gap-2 text-sm text-gray-600">
				Sort
				<select bind:value={sortBy} class="border border-gray-200 rounded-full px-3 py-2 text-sm bg-white">
					<option value="name">Name</option>
					<option value="size">Grid size</option>
				</select>
			</label>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-600 mb-6">
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-red-500 text-white text-[10px] font-bold leading-none">⌂</span>Home</span>
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-emerald-500 text-[9px] leading-none">🌳</span>Park</span>
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-yellow-400 text-[9px] leading-none">⚡</span>Charger</span>
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-blue-400 text-white text-[10px] font-bold leading-none">≈</span>Water</span>
		<span class="inline-flex items-center gap-1"><span class="h-4 w-4 rounded-sm bg-slate-700"></span>Blocked</span>
	</div>

	{#if $mapsResult.fetching && cards.length === 0}
		<div class="flex items-center justify-center py-24 text-gray-400 bg-white rounded-2xl border border-[#e8e8e8]">
			<div class="text-center"><span class="text-4xl mb-4 block">🗺️</span><p class="text-lg font-light">Loading maps…</p></div>
		</div>
	{:else if $mapsResult.error}
		<div class="rounded-2xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
			Could not load maps: {$mapsResult.error.message}
		</div>
	{:else if visible.length === 0}
		<div class="rounded-2xl border border-[#e8e8e8] bg-white py-16 text-center text-gray-500">
			<p class="text-lg font-light">No maps match “{search}”</p>
			<button type="button" onclick={() => (search = '')} class="mt-3 text-sm text-[#393c41] underline">Clear search</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
			{#each visible as map (map.mapId)}
				<article class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm overflow-hidden flex flex-col">
					<div class="p-4 border-b border-gray-100">
						<div class="flex items-start justify-between gap-3">
							<div>
								<h3 class="text-lg font-light text-[#393c41]">{map.name}</h3>
								<p class="text-xs font-mono text-gray-400 mt-0.5">{map.mapId}</p>
							</div>
							<span class="shrink-0 text-[11px] text-gray-400 bg-[#f7f7f7] rounded-full px-2 py-1">{map.gridSize}×{map.gridSize}</span>
						</div>
						<p class="text-sm text-gray-500 font-light leading-relaxed mt-3 line-clamp-2">{map.description}</p>
					</div>

					<div class="p-4 grow">
						{#if map.preview}
							<div class="overflow-hidden rounded-xl bg-[#f7f7f7] p-2">
								<div class="grid gap-0.5 aspect-square" style={`grid-template-columns: repeat(${map.preview.gridSize}, minmax(0, 1fr));`} aria-label={`Preview of ${map.name}`}>
									{#each map.preview.layout as row}
										{#each row.split('') as char}
											{@const t = tileType(char, map.preview.legend, map.preview.cellConfigs)}
											<div class={`aspect-square rounded-[2px] flex items-center justify-center text-[0.5rem] leading-none font-bold ${t === 'home' || t === 'water' ? 'text-white' : 'text-orange-500'} ${tileClass(char, map.preview.legend, map.preview.cellConfigs)}`} title={t}>{terrainGlyph(t) || directionGlyph(directionsForChar(char, map.preview.cellConfigs))}</div>
										{/each}
									{/each}
								</div>
							</div>
						{:else if map.error}
							<div class="flex items-center justify-center aspect-square rounded-xl bg-red-50 text-xs text-red-700 text-center p-4">Could not load preview</div>
						{:else}
							<div class="aspect-square rounded-xl bg-[#f7f7f7] animate-pulse"></div>
						{/if}
					</div>

					<div class="px-4 pb-4 mt-auto">
						<dl class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-600 mb-4">
							<div class="flex gap-1"><dt>🌳</dt><dd>{countType(map, 'park')} parks</dd></div>
							<div class="flex gap-1"><dt>⚡</dt><dd>{countType(map, 'supercharger')} chargers</dd></div>
							<div class="flex gap-1"><dt>🔋</dt><dd>{map.maxBattery} battery</dd></div>
						</dl>
						<div class="flex items-center justify-between gap-3 text-sm">
							<div class="flex items-center gap-3 text-xs">
								<a href={`/editor?map=${map.mapId}&duplicate=1`} class="font-medium text-gray-600 hover:text-[#393c41] transition-colors">Duplicate</a>
								<a href={`/editor?map=${map.mapId}`} class="font-medium text-gray-600 hover:text-[#393c41] transition-colors">Edit</a>
							</div>
							<a href={`/?map=${map.mapId}`} class="font-medium bg-[#393c41] text-white px-4 py-1.5 rounded-full hover:bg-black transition-colors">Use map →</a>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</div>
