<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { page } from '$app/stores';
	import { MAPS_QUERY } from '$lib/queries';
	import { fetchMapLayout } from '$lib/mapData';
	import { countTiles, prettyMapName, type MapLayout, type MapSummary } from '$lib/maps';
	import { MapSessionStarter } from '$lib/mapStart.svelte';
	import MapThumb from '$lib/components/MapThumb.svelte';
	import StartSessionModal from '$lib/components/StartSessionModal.svelte';

	const client = getContextClient();
	const mapsResult = queryStore({ client, query: gql(MAPS_QUERY) });

	const mapId = $derived(decodeURIComponent($page.params.id ?? ''));

	const summary = $derived<MapSummary | null>(
		($mapsResult?.data?.maps ?? []).find((m: MapSummary) => m.mapId === mapId) ?? null
	);

	const starter = new MapSessionStarter(client);

	// undefined = loading, null = failed.
	let layout = $state<MapLayout | null | undefined>(undefined);
	let copyState = $state<'idle' | 'copied' | 'failed'>('idle');

	$effect(() => {
		const id = mapId;
		if (!id) return;
		layout = undefined;
		void fetchMapLayout(client, id).then((l) => (layout = l));
	});

	const displayName = $derived(prettyMapName(summary?.name || layout?.name || mapId));
	const gridSize = $derived(summary?.gridSize ?? layout?.gridSize ?? 0);
	const maxBattery = $derived(summary?.maxBattery ?? layout?.maxBattery ?? 0);
	const startingBattery = $derived(layout?.startingBattery ?? 0);
	const description = $derived(summary?.description || layout?.description || '');

	const stats = $derived.by(() => {
		if (!layout) return [];
		return [
			{ label: 'Parks', value: countTiles(layout, 'park'), icon: '🌳' },
			{ label: 'Chargers', value: countTiles(layout, 'supercharger'), icon: '⚡' },
			{ label: 'Homes', value: countTiles(layout, 'home'), icon: '⌂' },
			{ label: 'Obstacles', value: countTiles(layout, 'water') + countTiles(layout, 'building'), icon: '▨' }
		];
	});

	// Only report "not found" once both the map list and the layout lookup have resolved.
	const notFound = $derived(
		!$mapsResult.fetching && $mapsResult.error === undefined && layout === null && summary === null
	);

	async function share() {
		const url = $page.url.href;
		try {
			await navigator.clipboard.writeText(url);
			copyState = 'copied';
		} catch {
			copyState = 'failed';
		}
		setTimeout(() => (copyState = 'idle'), 2500);
	}
</script>

<svelte:head>
	<title>{notFound ? 'Map not found' : `${displayName} — Tesla Road Trip`}</title>
	{#if !notFound && description}
		<meta name="description" content={description} />
	{/if}
</svelte:head>

<div class="bg-white border-b border-[#e8e8e8]">
	<div class="max-w-7xl mx-auto px-6 py-8">
		<a href="/maps" class="text-sm text-gray-500 hover:text-[#393c41]">← All maps</a>
		{#if notFound}
			<h1 class="text-3xl font-light text-[#171a20] mt-4">Map not found</h1>
			<p class="text-gray-500 mt-2">
				There is no map called <code class="text-[#393c41]">{mapId}</code>. It may have been renamed or deleted.
			</p>
			<a href="/maps" class="inline-block mt-6 bg-[#393c41] text-white text-sm px-5 py-2.5 rounded-full hover:bg-black transition-colors">Browse all maps</a>
		{:else}
			<div class="mt-4 flex flex-col lg:flex-row lg:items-end lg:justify-between gap-6">
				<div class="min-w-0">
					<p class="text-xs font-bold uppercase tracking-widest text-red-600 mb-3">🗺️ Map</p>
					<h1 class="text-3xl lg:text-5xl font-light text-[#171a20] tracking-tight">{displayName}</h1>
					{#if description}
						<p class="text-lg text-gray-500 font-light max-w-2xl mt-3">{description}</p>
					{/if}
				</div>
				<div class="flex flex-wrap gap-3">
					<button
						type="button"
						onclick={() => starter.open({ mapId, name: displayName, description, gridSize, maxBattery })}
						disabled={starter.starting || layout === undefined}
						class="bg-[#393c41] text-white text-sm px-6 py-3 rounded-full hover:bg-black transition-colors disabled:opacity-50"
					>{starter.starting ? 'Starting…' : 'Play this map'}</button>
					<button
						type="button"
						onclick={share}
						class="border border-gray-200 bg-white text-[#393c41] text-sm px-6 py-3 rounded-full hover:border-gray-400 transition-colors"
					>{copyState === 'copied' ? 'Link copied ✓' : copyState === 'failed' ? 'Copy failed' : 'Copy link'}</button>
				</div>
			</div>
		{/if}
	</div>
</div>

{#if !notFound}
	<div class="max-w-7xl mx-auto px-6 py-10">
		{#if $mapsResult.fetching && layout === undefined}
			<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
				<div class="aspect-square w-full rounded-2xl bg-white border border-[#e8e8e8] animate-pulse"></div>
				<div class="h-64 rounded-2xl bg-white border border-[#e8e8e8] animate-pulse"></div>
			</div>
		{:else}
			<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
				<section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-4 sm:p-6">
					{#if layout}
						<MapThumb map={layout} label={`Layout of ${displayName}`} />
					{:else}
						<div class="aspect-square w-full rounded-xl bg-red-50 flex items-center justify-center text-sm text-red-700 p-6 text-center">
							The layout for this map could not be loaded.
						</div>
					{/if}
				</section>

				<aside class="space-y-6">
					<section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
						<h2 class="text-lg font-light text-[#393c41] mb-4">Details</h2>
						<dl class="space-y-2 text-sm">
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-gray-500">Grid</dt>
								<dd class="text-[#393c41] tabular-nums">{gridSize ? `${gridSize}×${gridSize}` : '—'}</dd>
							</div>
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-gray-500">Battery</dt>
								<dd class="text-[#393c41] tabular-nums">{startingBattery || '—'} / {maxBattery || '—'}</dd>
							</div>
							<div class="flex items-baseline justify-between gap-3">
								<dt class="text-gray-500">Map ID</dt>
								<dd class="text-[#393c41] font-mono text-xs">{mapId}</dd>
							</div>
						</dl>
						{#if stats.length}
							<div class="mt-4 grid grid-cols-2 gap-2">
								{#each stats as stat}
									<div class="rounded-xl bg-[#f7f7f7] px-3 py-2">
										<p class="text-[11px] text-gray-500">{stat.icon} {stat.label}</p>
										<p class="text-base font-medium text-[#171a20] tabular-nums">{stat.value}</p>
									</div>
								{/each}
							</div>
						{/if}
					</section>

					<section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
						<h2 class="text-lg font-light text-[#393c41] mb-4">Legend</h2>
						<ul class="space-y-1.5 text-xs text-gray-600">
							<li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-red-500"></span>Home — charge to full</li>
							<li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-emerald-500"></span>Park — visit them all to win</li>
							<li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-yellow-400"></span>Supercharger — partial charge</li>
							<li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-blue-400"></span>Water — impassable</li>
							<li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-slate-700"></span>Blocked — impassable</li>
						</ul>
					</section>

					<section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6">
						<h2 class="text-lg font-light text-[#393c41] mb-2">Share this map</h2>
						<p class="text-xs text-gray-500">
							Send this page's link and your friend can start a session on {displayName} straight away.
						</p>
						<button
							type="button"
							onclick={share}
							class="mt-4 w-full border border-gray-200 bg-white text-[#393c41] text-sm px-4 py-2.5 rounded-full hover:border-gray-400 transition-colors"
						>{copyState === 'copied' ? 'Link copied ✓' : copyState === 'failed' ? 'Copy failed' : 'Copy link'}</button>
					</section>
				</aside>
			</div>
		{/if}
	</div>
{/if}

{#if starter.map}
	<StartSessionModal
		map={starter.map}
		starting={starter.starting}
		error={starter.error}
		onstart={(settings) => starter.begin(settings)}
		onclose={() => starter.cancel()}
	/>
{/if}
