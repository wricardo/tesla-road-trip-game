<script lang="ts">
	import { directionGlyph, directionsForChar, terrainGlyph } from '$lib/directional';
	import { tileClass, tileType, type MapLayout } from '$lib/maps';

	let { map, label }: { map: MapLayout; label: string } = $props();

	const rows = $derived(map.layout.length);
	const cols = $derived(Math.max(1, ...map.layout.map((r) => r.length)));
	// Glyphs become noise on big maps; colour alone carries the preview there.
	const showGlyphs = $derived(Math.max(rows, cols) <= 24);
</script>

<!-- Fixed square frame; the grid is scaled to fit inside it and centred. -->
<div class="aspect-square w-full rounded-xl bg-[#f7f7f7] p-2 flex items-center justify-center overflow-hidden">
	<div
		class="grid gap-px max-w-full max-h-full"
		style="grid-template-columns: repeat({cols}, minmax(0, 1fr)); aspect-ratio: {cols} / {rows}; width: {cols >= rows ? '100%' : 'auto'}; height: {rows > cols ? '100%' : 'auto'};"
		role="img"
		aria-label={label}
	>
		{#each map.layout as row}
			{#each row.split('') as char}
				{@const t = tileType(char, map.legend, map.cellConfigs)}
				<div class="aspect-square rounded-[1px] flex items-center justify-center text-[0.5rem] leading-none font-bold overflow-hidden {t === 'home' || t === 'water' ? 'text-white' : 'text-orange-500'} {tileClass(t)}">
					{#if showGlyphs}{terrainGlyph(t) || directionGlyph(directionsForChar(char, map.cellConfigs))}{/if}
				</div>
			{/each}
		{/each}
	</div>
</div>
