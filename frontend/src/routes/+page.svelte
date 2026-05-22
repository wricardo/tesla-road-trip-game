<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { MAPS_QUERY, MAP_QUERY, CREATE_SESSION_MUTATION, UPDATE_SESSION_MUTATION } from '$lib/queries';

	type LegendEntry = { key: string; value: string };
	type MapPreview = {
		name: string;
		description: string;
		gridSize: number;
		maxBattery: number;
		startingBattery: number;
		layout: string[];
		legend: LegendEntry[];
	};

	const client = getContextClient();
	const mapsResult = queryStore({ client, query: gql(MAPS_QUERY) });
	const maps = $derived($mapsResult?.data?.maps ?? []);

	let showCreate = $state(false);
	let selectedMap = $state($page.url.searchParams.get('map') ?? '');
	let sessionName = $state('');
	let createError = $state('');
	let creating = $state(false);
	let preview = $state<MapPreview | null>(null);
	let previewError = $state('');
	let previewLoading = $state(false);
	const previewMapID = $derived(selectedMap || maps.find((m: { mapId: string }) => m.mapId === 'classic')?.mapId || maps[0]?.mapId || '');
	let previewRequest = 0;

	$effect(() => {
		const mapID = previewMapID;
		if (!mapID) {
			preview = null;
			previewError = '';
			return;
		}

		const requestID = ++previewRequest;
		previewLoading = true;
		previewError = '';
		client.query(gql(MAP_QUERY), { name: mapID }).toPromise().then((result) => {
			if (requestID !== previewRequest) return;
			previewLoading = false;
			if (result.error) {
				preview = null;
				previewError = result.error.message;
				return;
			}
			preview = result.data?.map ?? null;
		});
	});

	function tileType(char: string, legend: LegendEntry[] = []) {
		return legend.find((entry) => entry.key === char)?.value ?? 'road';
	}

	function tileClass(char: string, legend: LegendEntry[] = []) {
		switch (tileType(char, legend)) {
			case 'home': return 'bg-red-500 ring-2 ring-red-200';
			case 'park': return 'bg-emerald-500';
			case 'supercharger': return 'bg-yellow-400';
			case 'water': return 'bg-blue-400';
			case 'building': return 'bg-slate-700';
			default: return 'bg-white';
		}
	}

	async function createSession() {
		if (creating) return;
		creating = true;
		createError = '';
		const result = await client.mutation(gql(CREATE_SESSION_MUTATION), { mapID: selectedMap || null }).toPromise();
		creating = false;
		if (result.error) { createError = result.error.message; return; }
		const id = result.data?.createSession?.id;
		if (id) {
			const name = sessionName.trim();
			if (name) {
				await client.mutation(gql(UPDATE_SESSION_MUTATION), { id, displayName: name }).toPromise();
			}
			goto(`/watch/${id}`);
		}
	}
</script>

<svelte:head>
	<title>Tesla Road Trip Game</title>
</svelte:head>

<!-- Hero -->
<div class="bg-white border-b border-[#e8e8e8]">
	<div class="max-w-7xl mx-auto px-6 py-16 lg:py-20">
		<div class="lg:grid lg:grid-cols-[1fr_380px] gap-12 items-start">
			<!-- Left: pitch -->
			<div>
				<p class="text-xs font-bold uppercase tracking-widest text-red-500 mb-4">🚗 Learn AI through interactive gameplay</p>
				<h1 class="text-4xl lg:text-5xl font-light text-[#171a20] leading-tight tracking-tight mb-6">
					Teach your AI agent<br>to drive a Tesla.<br>Manage resources. Navigate maps.
				</h1>
				<p class="text-lg text-gray-500 font-light leading-relaxed max-w-2xl mb-10">
						An educational game where AI agents learn pathfinding, resource management, and decision-making. <a href="/learn" class="text-red-500 font-medium hover:underline">Learn more</a>
				</p>

				<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
					<div class="bg-[#f7f7f7] rounded-2xl p-5">
						<div class="text-2xl mb-2">🗺️</div>
						<div class="font-medium text-[#393c41] mb-1">Choose a map</div>
						<p class="text-sm text-gray-400 leading-relaxed">Start with a built-in configuration or design your own custom road trip.</p>
					</div>
					<div class="bg-[#f7f7f7] rounded-2xl p-5">
						<div class="text-2xl mb-2">⚡</div>
						<div class="font-medium text-[#393c41] mb-1">Spend energy wisely</div>
						<p class="text-sm text-gray-400 leading-relaxed">Every move drains battery. Route through chargers before you get stranded.</p>
					</div>
					<div class="bg-[#f7f7f7] rounded-2xl p-5">
						<div class="text-2xl mb-2">🤖</div>
						<div class="font-medium text-[#393c41] mb-1">Built for AI play</div>
						<p class="text-sm text-gray-400 leading-relaxed">Use GraphQL, WebSockets, and llms.txt to let agents inspect and control sessions.</p>
					</div>
				</div>
			</div>

			<!-- Right: quick actions -->
			<div class="mt-10 lg:mt-0 bg-[#f7f7f7] rounded-2xl p-6 border border-[#e8e8e8]">
				<h2 class="text-xl font-light text-[#393c41] mb-1">Start a road trip</h2>
				<p class="text-sm text-gray-400 mb-5">Create a session, open the tools, or let an AI drive.</p>

				<div class="mb-4">
					<label for="cfg" class="block text-xs font-semibold text-[#393c41] mb-1.5">Map</label>
					<select id="cfg" bind:value={selectedMap}
						class="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm bg-white focus:outline-none focus:border-gray-400">
						<option value="">Default</option>
						{#each maps as m}
							<option value={m.mapId}>{m.name}</option>
						{/each}
					</select>
				</div>

				<div class="mb-4">
					<label for="session-name" class="block text-xs font-semibold text-[#393c41] mb-1.5">Name <span class="font-normal text-gray-400">(optional)</span></label>
					<input
						id="session-name"
						type="text"
						bind:value={sessionName}
						placeholder="e.g. Claude's first run"
						class="w-full border border-gray-200 rounded-xl px-3 py-2.5 text-sm bg-white focus:outline-none focus:border-gray-400"
					/>
				</div>

				<div class="mb-5 rounded-2xl border border-gray-200 bg-white p-3">
					<div class="flex items-start justify-between gap-3 mb-3">
						<div>
							<p class="text-xs font-semibold uppercase tracking-wide text-gray-400">Map preview</p>
							<p class="text-sm font-medium text-[#393c41]">{preview?.name ?? 'Loading map…'}</p>
						</div>
						{#if preview}
							<span class="shrink-0 text-[11px] text-gray-400 bg-[#f7f7f7] rounded-full px-2 py-1">{preview.gridSize}×{preview.gridSize}</span>
						{/if}
					</div>

					{#if previewLoading && !preview}
						<div class="h-44 rounded-xl bg-[#f7f7f7] animate-pulse"></div>
					{:else if previewError}
						<p class="text-xs text-red-500">Could not load preview: {previewError}</p>
					{:else if preview}
						<div class="overflow-hidden rounded-xl bg-[#f7f7f7] p-2">
							<div class="grid gap-0.5 aspect-square" style={`grid-template-columns: repeat(${preview.gridSize}, minmax(0, 1fr));`} aria-label={`Preview of ${preview.name}`}>
								{#each preview.layout as row}
									{#each row.split('') as char}
										<div class={`aspect-square rounded-[2px] ${tileClass(char, preview.legend)}`} title={tileType(char, preview.legend)}></div>
									{/each}
								{/each}
							</div>
						</div>
						<div class="mt-3 flex flex-wrap gap-2 text-[11px] text-gray-500">
							<span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-red-500"></span>Home</span>
							<span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-emerald-500"></span>Park</span>
							<span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-yellow-400"></span>Charger</span>
							<span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-slate-700"></span>Blocked</span>
							<span class="inline-flex items-center gap-1"><span class="h-2.5 w-2.5 rounded-sm bg-blue-400"></span>Water</span>
						</div>
					{/if}
				</div>

				<button
					onclick={createSession}
					disabled={creating}
					class="w-full bg-[#393c41] text-white text-sm px-4 py-3 rounded-full hover:bg-black transition-colors disabled:opacity-50 mb-3">
					{creating ? 'Creating…' : '+ Create session'}
				</button>

				<a href="/lobby" class="block w-full text-center border border-gray-200 bg-white text-[#393c41] text-sm px-4 py-3 rounded-full hover:border-gray-400 transition-colors">
					Watch live sessions
				</a>

				{#if createError}
					<p class="text-xs text-red-500 mt-3">{createError}</p>
				{/if}
			</div>
		</div>
	</div>
</div>

