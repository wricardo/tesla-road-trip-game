<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { goto } from '$app/navigation';
	import { MAPS_QUERY, CREATE_SESSION_MUTATION } from '$lib/queries';

	const client = getContextClient();
	const mapsResult = queryStore({ client, query: gql(MAPS_QUERY) });
	const maps = $derived($mapsResult?.data?.maps ?? []);

	let showCreate = $state(false);
	let selectedMap = $state('');
	let createError = $state('');
	let creating = $state(false);

	async function createSession() {
		if (creating) return;
		creating = true;
		createError = '';
		const result = await client.mutation(gql(CREATE_SESSION_MUTATION), { mapID: selectedMap || null }).toPromise();
		creating = false;
		if (result.error) { createError = result.error.message; return; }
		const id = result.data?.createSession?.id;
		if (id) goto(`/watch/${id}`);
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

