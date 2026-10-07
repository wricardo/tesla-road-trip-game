<script lang="ts">
	let {
		won,
		title,
		detail,
		moves,
		parks,
		totalParks,
		battery,
		maxBattery,
		resets,
		nextMapName = null,
		resetting = false,
		startingNext = false,
		nextError = null,
		onreset,
		onnext,
		onclose
	}: {
		won: boolean;
		title: string;
		detail: string;
		moves: number;
		parks: number;
		totalParks: number;
		battery: number;
		maxBattery: number;
		resets: number;
		nextMapName?: string | null;
		resetting?: boolean;
		startingNext?: boolean;
		nextError?: string | null;
		onreset: () => void;
		onnext: () => void;
		onclose: () => void;
	} = $props();

	let primary = $state<HTMLButtonElement | null>(null);
	$effect(() => {
		primary?.focus();
	});
</script>

<div class="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
	<div role="dialog" aria-modal="true" aria-labelledby="result-title" class="relative w-full max-w-md rounded-2xl bg-white p-6 shadow-xl text-center" data-testid="result-modal">
		<button type="button" onclick={onclose} class="absolute right-4 top-3 text-gray-400 hover:text-gray-900" aria-label="Close and keep the board visible">✕</button>
		<h2 id="result-title" class="text-2xl font-light {won ? 'text-green-800' : 'text-red-700'}">{title}</h2>
		<p class="mt-2 text-sm text-gray-600">{detail}</p>
		<dl class="mt-5 grid grid-cols-4 gap-2 text-center">
			<div class="rounded-xl bg-gray-50 px-2 py-2"><dt class="text-[11px] text-gray-500">Parks</dt><dd class="text-base font-medium text-gray-800">{parks} / {totalParks}</dd></div>
			<div class="rounded-xl bg-gray-50 px-2 py-2"><dt class="text-[11px] text-gray-500">Moves</dt><dd class="text-base font-medium text-gray-800">{moves}</dd></div>
			<div class="rounded-xl bg-gray-50 px-2 py-2"><dt class="text-[11px] text-gray-500">Battery</dt><dd class="text-base font-medium text-gray-800">{battery} / {maxBattery}</dd></div>
			<div class="rounded-xl bg-gray-50 px-2 py-2"><dt class="text-[11px] text-gray-500">Resets</dt><dd class="text-base font-medium text-gray-800">{resets}</dd></div>
		</dl>
		<div class="mt-6 flex flex-wrap justify-center gap-2">
			<button
				bind:this={primary}
				type="button"
				onclick={onreset}
				disabled={resetting}
				class="text-sm px-5 py-2.5 rounded-full bg-[#393c41] text-white hover:bg-black transition-colors disabled:opacity-50"
			>{resetting ? 'Resetting…' : 'Try again'}</button>
			<button
				type="button"
				onclick={onnext}
				disabled={!nextMapName || startingNext}
				title={nextMapName ? `Start a new session on ${nextMapName}` : 'No other map available'}
				class="text-sm px-5 py-2.5 rounded-full border border-gray-300 text-[#393c41] hover:border-gray-500 transition-colors disabled:opacity-50"
			>{startingNext ? 'Starting…' : 'Next map'}</button>
			<a href="/maps" class="text-sm px-5 py-2.5 rounded-full border border-gray-300 text-[#393c41] hover:border-gray-500 transition-colors">All maps</a>
		</div>
		{#if nextMapName}
			<p class="mt-3 text-xs text-gray-500">Next map: {nextMapName}</p>
		{/if}
		{#if nextError}
			<p class="mt-2 text-xs text-red-700">{nextError}</p>
		{/if}
	</div>
</div>
