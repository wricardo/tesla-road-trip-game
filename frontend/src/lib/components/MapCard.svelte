<script lang="ts">
	import MapThumb from './MapThumb.svelte';
	import { countTiles, prettyMapName, type MapLayout, type MapSummary } from '$lib/maps';

	let {
		map,
		preview = undefined,
		previewError = false,
		busy = false,
		onplay,
		showAdminLinks = false
	}: {
		map: MapSummary;
		preview?: MapLayout;
		previewError?: boolean;
		busy?: boolean;
		onplay: (map: MapSummary) => void;
		showAdminLinks?: boolean;
	} = $props();

	const name = $derived(prettyMapName(map.name || map.mapId));
	const parks = $derived(preview ? countTiles(preview, 'park') : null);
</script>

<article class="h-full bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-4 flex flex-col gap-3" data-testid="map-card">
	{#if preview}
		<MapThumb map={preview} label={`Preview of ${name}`} />
	{:else if previewError}
		<div class="aspect-square w-full rounded-xl bg-red-50 flex items-center justify-center text-xs text-red-700 p-4 text-center">Could not load preview</div>
	{:else}
		<div class="aspect-square w-full rounded-xl bg-[#f7f7f7] animate-pulse"></div>
	{/if}
	<div class="min-w-0">
		<h3 class="text-lg font-light text-[#171a20] truncate" title={map.mapId}>{name}</h3>
		<p class="text-sm text-gray-500 font-light leading-relaxed line-clamp-2 min-h-[2.5rem]">{map.description}</p>
	</div>
	<div class="flex flex-wrap gap-1.5 text-xs">
		<span class="rounded-full bg-[#f7f7f7] text-[#393c41] px-2 py-0.5">{map.gridSize}×{map.gridSize}</span>
		<span class="rounded-full bg-emerald-50 text-emerald-800 px-2 py-0.5">🌳 {parks ?? '…'} parks</span>
		<span class="rounded-full bg-[#f7f7f7] text-[#393c41] px-2 py-0.5">🔋 ≤ {map.maxBattery} battery</span>
	</div>
	<div class="mt-auto flex items-center gap-2">
		<button
			type="button"
			onclick={() => onplay(map)}
			disabled={busy}
			class="flex-1 bg-[#393c41] text-white text-sm px-4 py-2 rounded-full hover:bg-black transition-colors disabled:opacity-50"
		>{busy ? 'Starting…' : 'Play'}</button>
		{#if showAdminLinks}
			<a href={`/editor?map=${map.mapId}`} class="text-xs text-gray-600 border border-gray-200 rounded-full px-3 py-2 hover:border-gray-400 transition-colors">Edit</a>
			<a href={`/editor?map=${map.mapId}&duplicate=1`} class="text-xs text-gray-600 border border-gray-200 rounded-full px-3 py-2 hover:border-gray-400 transition-colors">Duplicate</a>
		{/if}
	</div>
</article>
