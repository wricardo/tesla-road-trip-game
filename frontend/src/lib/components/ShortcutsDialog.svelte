<script lang="ts">
	let { shortcuts, onclose }: { shortcuts: { keys: string[]; action: string }[]; onclose: () => void } = $props();

	let closeBtn = $state<HTMLButtonElement | null>(null);
	$effect(() => {
		closeBtn?.focus();
	});
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
	<div role="dialog" aria-modal="true" aria-labelledby="shortcuts-title" class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl">
		<div class="flex items-center justify-between mb-4">
			<h2 id="shortcuts-title" class="text-lg font-light text-[#171a20]">Keyboard shortcuts</h2>
			<button bind:this={closeBtn} type="button" onclick={onclose} class="text-sm text-gray-500 hover:text-gray-900" aria-label="Close shortcuts">✕</button>
		</div>
		<dl class="space-y-2 text-sm">
			{#each shortcuts as s}
				<div class="flex items-center justify-between gap-4">
					<dt class="flex gap-1">{#each s.keys as k}<kbd class="inline-flex min-w-[1.5rem] h-6 items-center justify-center rounded-md border border-gray-300 bg-gray-50 px-1.5 text-xs font-semibold text-gray-700">{k}</kbd>{/each}</dt>
					<dd class="text-gray-600 text-right">{s.action}</dd>
				</div>
			{/each}
		</dl>
		<p class="mt-4 text-xs text-gray-500">Esc closes this dialog.</p>
	</div>
</div>
