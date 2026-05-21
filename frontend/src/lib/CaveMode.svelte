<script lang="ts">
	import { browser } from '$app/environment';

	let { enabled = $bindable(false), radius = $bindable(3) } = $props();

	// persist in localStorage
	$effect(() => {
		if (!browser) return;
		const saved = localStorage.getItem('caveMode');
		if (saved) {
			const p = JSON.parse(saved);
			enabled = p.enabled ?? false;
			radius = p.radius ?? 3;
		}
	});

	$effect(() => {
		if (!browser) return;
		localStorage.setItem('caveMode', JSON.stringify({ enabled, radius }));
	});
</script>

<div class="mt-4 bg-white rounded-xl border border-[#e8e8e8] px-4 py-3 flex items-center gap-4">
	<label class="flex items-center gap-2 cursor-pointer select-none">
		<input type="checkbox" bind:checked={enabled} class="sr-only peer" />
		<div class="w-8 h-4 bg-gray-200 peer-checked:bg-[#393c41] rounded-full relative transition-colors">
			<span class="absolute top-0.5 left-0.5 w-3 h-3 bg-white rounded-full shadow transition-transform peer-checked:translate-x-4 {enabled ? 'translate-x-4' : ''}"></span>
		</div>
		<span class="text-xs text-gray-500">Cave Mode</span>
	</label>

	{#if enabled}
		<div class="flex items-center gap-2 flex-1">
			<span class="text-xs text-gray-400">Radius</span>
			<input type="range" min="1" max="10" bind:value={radius} class="flex-1 h-1 accent-[#393c41]" />
			<span class="text-xs text-gray-500 w-4 text-right">{radius}</span>
		</div>
		<p class="hidden text-xs text-gray-400 italic">Fog is your view only — not the AI's</p>
	{/if}
</div>
