<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { goto } from '$app/navigation';
	import { MAPS_QUERY } from '$lib/queries';
	import { prettyMapName, sortMaps, type MapLayout, type MapSummary } from '$lib/maps';
	import { fetchMapLayout } from '$lib/mapData';
	import { createGameSession } from '$lib/createSession';
	import MapCard from '$lib/components/MapCard.svelte';

	const client = getContextClient();
	const mapsResult = queryStore({ client, query: gql(MAPS_QUERY) });

	const maps = $derived<MapSummary[]>(sortMaps($mapsResult?.data?.maps ?? []));

	// undefined = loading, null = failed.
	let previews = $state<Record<string, MapLayout | null>>({});
	const requested = new Set<string>();
	$effect(() => {
		for (const m of maps) {
			if (requested.has(m.mapId)) continue;
			requested.add(m.mapId);
			void fetchMapLayout(client, m.mapId).then((p) => (previews[m.mapId] = p));
		}
	});
	const loadingPreviews = $derived(maps.some((m) => !(m.mapId in previews)));

	let search = $state('');
	const visible = $derived.by(() => {
		const q = search.trim().toLowerCase();
		return maps.filter((m) => !q || `${prettyMapName(m.name)} ${m.name} ${m.mapId} ${m.description}`.toLowerCase().includes(q));
	});

	let creatingId = $state<string | null>(null);
	let createError = $state('');
	async function play(map: MapSummary) {
		if (creatingId) return;
		creatingId = map.mapId;
		createError = '';
		try {
			goto(`/watch/${await createGameSession(client, { mapID: map.mapId })}`);
		} catch (err) {
			createError = err instanceof Error ? err.message : 'Could not create the session';
		} finally {
			creatingId = null;
		}
	}
</script>

<svelte:head>
	<title>Maps — Tesla Road Trip</title>
</svelte:head>

<div class="bg-white border-b border-[#e8e8e8]">
	<div class="max-w-7xl mx-auto px-6 py-10 lg:py-12">
		<p class="text-xs font-bold uppercase tracking-widest text-red-600 mb-4">🗺️ Map gallery</p>
		<div class="flex flex-col lg:flex-row lg:items-end lg:justify-between gap-6">
			<div>
				<h1 class="text-4xl lg:text-5xl font-light text-[#171a20] tracking-tight mb-4">Choose your road trip</h1>
				<p class="text-lg text-gray-500 font-light max-w-2xl">
					Compare routes, chargers, parks and obstacles at a glance, then press Play to start a session.
				</p>
			</div>
			<div class="flex flex-wrap gap-3">
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
				{visible.length}{visible.length === maps.length ? '' : ` of ${maps.length}`} maps · smallest first{#if loadingPreviews} · loading previews…{/if}
			</p>
		</div>
		<input
			type="search"
			bind:value={search}
			placeholder="Search name or description"
			aria-label="Search maps"
			class="min-w-[16rem] border border-gray-200 rounded-full px-4 py-2 text-sm bg-white"
		/>
	</div>

	<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-600 mb-6">
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-red-500 text-white text-[10px] font-bold leading-none">⌂</span>Home</span>
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-emerald-500 text-[9px] leading-none">🌳</span>Park</span>
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-yellow-400 text-[9px] leading-none">⚡</span>Charger</span>
		<span class="inline-flex items-center gap-1"><span class="inline-flex h-4 w-4 items-center justify-center rounded-sm bg-blue-400 text-white text-[10px] font-bold leading-none">≈</span>Water</span>
		<span class="inline-flex items-center gap-1"><span class="h-4 w-4 rounded-sm bg-slate-700"></span>Blocked</span>
	</div>

	{#if createError}
		<p class="mb-4 rounded-2xl border border-red-200 bg-red-50 px-5 py-3 text-sm text-red-700">{createError}</p>
	{/if}

	{#if $mapsResult.fetching && maps.length === 0}
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
			<button type="button" onclick={() => (search = '')} class="mt-4 bg-[#393c41] text-white text-sm px-5 py-2.5 rounded-full hover:bg-black transition-colors">Clear search</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
			{#each visible as map (map.mapId)}
				<MapCard
					{map}
					preview={previews[map.mapId] ?? undefined}
					previewError={previews[map.mapId] === null}
					busy={creatingId === map.mapId}
					onplay={play}
					showAdminLinks
				/>
			{/each}
		</div>
	{/if}
</div>
