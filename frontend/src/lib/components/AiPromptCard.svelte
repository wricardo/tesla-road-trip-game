<script lang="ts">
	let { prompt, blurb }: { prompt: string; blurb: string } = $props();

	let copied = $state(false);
	let failed = $state(false);
	let open = $state(false);
	let area = $state<HTMLTextAreaElement | null>(null);
	let timer: ReturnType<typeof setTimeout> | undefined;

	async function copy() {
		failed = false;
		try {
			await navigator.clipboard.writeText(prompt);
			copied = true;
			clearTimeout(timer);
			timer = setTimeout(() => (copied = false), 2000);
		} catch {
			// Clipboard API unavailable (insecure context / permissions): show the text selected instead.
			failed = true;
			open = true;
			queueMicrotask(() => area?.select());
		}
	}
</script>

<div class="rounded-2xl bg-white border border-[#e8e8e8] shadow-sm p-4" data-testid="ai-prompt-card">
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<h2 class="text-xs font-semibold uppercase tracking-widest text-gray-600">Play with an AI</h2>
			<p class="mt-1 text-sm text-gray-700">{blurb}</p>
		</div>
		<button
			type="button"
			onclick={copy}
			class="text-sm px-4 py-2 rounded-full border transition-colors shrink-0 {copied ? 'bg-green-50 border-green-300 text-green-800' : 'border-gray-300 text-[#393c41] hover:bg-gray-50'}"
		>{copied ? 'Copied!' : 'Copy prompt'}</button>
	</div>
	<p class="sr-only" aria-live="polite">{copied ? 'Prompt copied to clipboard' : ''}</p>
	{#if failed}
		<p role="alert" class="mt-2 text-xs text-amber-800">Could not access the clipboard — the prompt is selected below, press Ctrl/Cmd+C.</p>
	{/if}
	<details class="mt-3" bind:open>
		<summary class="cursor-pointer text-xs font-medium text-gray-600 hover:text-gray-900 select-none">Show full prompt</summary>
		<textarea
			bind:this={area}
			readonly
			rows="16"
			aria-label="Prompt for an AI"
			value={prompt}
			class="mt-2 w-full text-xs font-mono text-gray-700 bg-gray-50 rounded-xl p-3 resize-y leading-relaxed border border-gray-100"
			onclick={(e) => (e.currentTarget as HTMLTextAreaElement).select()}
		></textarea>
	</details>
</div>
