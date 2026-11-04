<script lang="ts">
	import { DEFAULT_MOVE_DELAY_MS } from '$lib/createSession';
	import { prettyMapName, type MapSummary } from '$lib/maps';
	import {
		DEFAULT_START_SETTINGS,
		validateStartSettings,
		type StartSettings
	} from '$lib/startSettings';

	let {
		map,
		starting = false,
		error = null,
		onstart,
		onclose
	}: {
		map: MapSummary;
		starting?: boolean;
		error?: string | null;
		onstart: (settings: StartSettings) => void;
		onclose: () => void;
	} = $props();

	// A fresh modal always opens with the documented defaults (fog on, radius 1).
	let fogEnabled = $state(DEFAULT_START_SETTINGS.fogEnabled);
	let fogRadius = $state(DEFAULT_START_SETTINGS.fogRadius);
	let gridPassword = $state(DEFAULT_START_SETTINGS.gridPassword);
	let moveDelayMs = $state(DEFAULT_START_SETTINGS.moveDelayMs);
	let formError = $state<string | null>(null);

	let primary = $state<HTMLButtonElement | null>(null);
	$effect(() => {
		primary?.focus();
	});

	const mapName = $derived(prettyMapName(map.name || map.mapId));

	function submit(event: SubmitEvent) {
		event.preventDefault();
		if (starting) return;
		const settings: StartSettings = { fogEnabled, fogRadius, gridPassword, moveDelayMs };
		const problem = validateStartSettings(settings);
		if (problem) {
			formError = problem;
			return;
		}
		formError = null;
		onstart(settings);
	}

	function onkeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			onclose();
		}
	}
</script>

<svelte:window {onkeydown} />

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
	role="presentation"
	onclick={(e) => e.target === e.currentTarget && onclose()}
>
	<div
		role="dialog"
		aria-modal="true"
		aria-labelledby="start-title"
		class="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl"
		data-testid="start-modal"
	>
		<form onsubmit={submit} novalidate>
		<div class="flex items-start justify-between gap-4 mb-1">
			<div>
				<h2 id="start-title" class="text-xl font-light text-[#171a20]">Start {mapName}</h2>
				<p class="text-sm text-gray-500 mt-1">Choose how the session hides the map from AI agents.</p>
			</div>
			<button type="button" onclick={onclose} class="text-gray-400 hover:text-gray-900" aria-label="Close">✕</button>
		</div>

		<fieldset class="mt-5 rounded-2xl border border-gray-200 p-4">
			<legend class="px-1 text-xs font-semibold uppercase tracking-wide text-gray-500">Fog mode</legend>
			<label class="inline-flex items-center gap-2 text-sm text-[#393c41]">
				<input type="checkbox" bind:checked={fogEnabled} class="rounded border-gray-300" />
				Hide the map beyond a small radius
			</label>

			{#if fogEnabled}
				<div class="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-3">
					<div>
						<label for="start-fog-radius" class="block text-xs font-semibold text-[#393c41] mb-1.5">Radius</label>
						<input
							id="start-fog-radius"
							type="number"
							min="1"
							step="1"
							bind:value={fogRadius}
							class="w-full border border-gray-200 rounded-xl px-3 py-2 text-sm bg-white focus:outline-none focus:border-gray-400"
						/>
					</div>
					<div>
						<label for="start-grid-password" class="block text-xs font-semibold text-[#393c41] mb-1.5">Grid password</label>
						<input
							id="start-grid-password"
							type="text"
							bind:value={gridPassword}
							placeholder="leave blank to auto-generate"
							class="w-full border border-gray-200 rounded-xl px-3 py-2 text-sm bg-white focus:outline-none focus:border-gray-400"
						/>
					</div>
				</div>
				<p class="mt-2 text-xs text-gray-500">
					Agents only see cells within {fogRadius} cell{fogRadius === 1 ? '' : 's'} of the car. The full grid needs the
					password via GraphQL <code>grid(password: …)</code>.
				</p>
			{:else}
				<p class="mt-2 text-xs text-gray-500">Agents can read the whole map from the start.</p>
			{/if}
		</fieldset>

		<div class="mt-3">
			<label for="start-move-delay" class="block text-xs font-semibold uppercase tracking-wide text-gray-500 mb-1.5">Move delay (ms)</label>
			<input
				id="start-move-delay"
				type="number"
				min="0"
				step="1"
				bind:value={moveDelayMs}
				class="w-full border border-gray-200 rounded-xl px-3 py-2 text-sm bg-white focus:outline-none focus:border-gray-400"
			/>
			<p class="mt-2 text-xs text-gray-500">
				Pause between moves on the live view so spectators can follow fast AI moves. {DEFAULT_MOVE_DELAY_MS} ms is the
				default; 0 means no delay.
			</p>
		</div>

		{#if formError ?? error}
			<p class="mt-4 text-sm text-red-700" role="alert">{formError ?? error}</p>
		{/if}

		<div class="mt-6 flex justify-end gap-2">
			<button
				type="button"
				onclick={onclose}
				class="text-sm px-5 py-2.5 rounded-full border border-gray-300 text-[#393c41] hover:border-gray-500 transition-colors"
			>Cancel</button>
			<button
				bind:this={primary}
				type="submit"
				disabled={starting}
				class="text-sm px-5 py-2.5 rounded-full bg-[#393c41] text-white hover:bg-black transition-colors disabled:opacity-50"
			>{starting ? 'Starting…' : 'Start session'}</button>
		</div>
		</form>
	</div>
</div>
