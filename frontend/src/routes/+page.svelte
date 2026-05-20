<script lang="ts">
	import { getContextClient, queryStore, subscriptionStore, mutationStore, gql } from '@urql/svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { SESSIONS_QUERY, CONFIGS_QUERY, CREATE_SESSION_MUTATION } from '$lib/queries';

	const client = getContextClient();

	// initial sessions load
	const sessionsResult = queryStore({ client, query: gql(SESSIONS_QUERY) });
	const configsResult = queryStore({ client, query: gql(CONFIGS_QUERY) });

	type SessionCard = {
		id: string;
		configName: string;
		battery: number;
		maxBattery: number;
		score: number;
		victory: boolean;
		gameOver: boolean;
		totalMoves: number;
	};

	let sessionMap = $state<Map<string, SessionCard>>(new Map());

	// seed from query
	$effect(() => {
		const data = $sessionsResult?.data?.sessions?.sessions;
		if (!data) return;
		const m = new Map<string, SessionCard>();
		for (const s of data) {
			m.set(s.id, {
				id: s.id,
				configName: s.configName,
				battery: s.gameState.battery,
				maxBattery: s.gameState.maxBattery,
				score: s.gameState.score,
				victory: s.gameState.victory,
				gameOver: s.gameState.gameOver,
				totalMoves: s.gameState.totalMoves,
			});
		}
		sessionMap = m;
	});

	// live lobby updates — lobbyUpdated emits GameState with configName embedded
	// We need sessionId too; the subscription returns GameState which has configName but not sessionId.
	// Use sessionUpdated per-session instead; lobby just polls sessions query every 10s.
	let pollInterval: ReturnType<typeof setInterval>;
	onMount(() => {
		pollInterval = setInterval(() => {
			sessionsResult.reexecute?.({ requestPolicy: 'network-only' });
		}, 10_000);
		return () => clearInterval(pollInterval);
	});

	const sessions = $derived(Array.from(sessionMap.values()));

	// create session modal
	let showCreate = $state(false);
	let selectedConfig = $state('');
	let createdSession = $state<{id: string; configName: string} | null>(null);
	let createError = $state('');
	let creating = $state(false);

	async function createSession() {
		if (creating) return;
		creating = true;
		createError = '';
		const result = await client.mutation(gql(CREATE_SESSION_MUTATION), { configName: selectedConfig || null }).toPromise();
		creating = false;
		if (result.error) {
			createError = result.error.message;
			return;
		}
		const id = result.data?.createSession?.id;
		if (id) goto(`/watch/${id}`);
	}

	function resetModal() {
		showCreate = false;
		createdSession = null;
		createError = '';
		selectedConfig = '';
	}

	const configs = $derived($configsResult?.data?.configs ?? []);
</script>

<div class="max-w-7xl mx-auto px-6 py-10">
	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-3xl font-light text-[#393c41]">Live Sessions</h1>
			<p class="text-sm text-gray-400 mt-1">Watch AI agents navigate the grid in real time</p>
		</div>
		<button
			onclick={() => showCreate = true}
			class="bg-[#393c41] text-white text-sm px-5 py-2 rounded-full hover:bg-black transition-colors"
		>
			+ New Session
		</button>
	</div>

	{#if $sessionsResult.fetching && sessions.length === 0}
		<div class="flex items-center justify-center py-32 text-gray-400">
			<div class="text-center">
				<span class="text-4xl mb-4 block">🚗</span>
				<p class="text-lg font-light">Loading sessions…</p>
			</div>
		</div>
	{:else if sessions.length === 0}
		<div class="flex flex-col items-center justify-center py-32 text-gray-400">
			<span class="text-5xl mb-4">🚗</span>
			<p class="text-lg font-light">No active sessions</p>
			<p class="text-sm mt-2">Create one and point your AI at it</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
			{#each sessions as s (s.id)}
				<a href="/watch/{s.id}" class="block bg-white rounded-2xl shadow-sm border border-[#e8e8e8] p-5 hover:shadow-md transition-shadow">
					<div class="flex items-center justify-between mb-3">
						<span class="font-mono text-sm text-[#393c41]">{s.id}</span>
						<span class="text-xs text-gray-400 bg-gray-100 rounded-full px-2 py-0.5">{s.configName}</span>
					</div>
					<div class="mb-3">
						<div class="flex justify-between text-xs text-gray-400 mb-1">
							<span>Battery</span>
							<span>{s.battery}/{s.maxBattery}</span>
						</div>
						<div class="h-1.5 bg-gray-100 rounded-full overflow-hidden">
							<div
								class="h-full rounded-full transition-all {s.battery / s.maxBattery > 0.5 ? 'bg-green-400' : s.battery / s.maxBattery > 0.25 ? 'bg-orange-400' : 'bg-red-400'}"
								style="width: {Math.max(0, (s.battery / s.maxBattery) * 100)}%"
							></div>
						</div>
					</div>
					<div class="flex gap-4 text-xs text-gray-500">
						<span>🌳 {s.score}</span>
						<span>📍 {s.totalMoves} moves</span>
						<span class="ml-auto {s.victory ? 'text-green-500' : s.gameOver ? 'text-red-500' : 'text-gray-400'}">
							{s.victory ? '🏆 Won' : s.gameOver ? '💥 Crashed' : '🟢 Active'}
						</span>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>

{#if showCreate}
	<div
		class="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
		role="dialog"
		aria-modal="true"
		aria-label="Create new session"
	>
		<div class="bg-white rounded-2xl shadow-xl p-8 w-full max-w-md mx-4">
			<h2 class="text-xl font-light text-[#393c41] mb-6">New Session</h2>

			{#if createdSession}
				<div>
					<p class="text-sm text-gray-500 mb-3">Session <code class="font-mono">{createdSession.id}</code> ready. Point your AI here:</p>
					<pre class="bg-gray-50 rounded-lg p-3 text-xs font-mono text-gray-700 mb-2 overflow-x-auto">curl -X POST /graphql \
  -H 'Content-Type: application/json' \
  -d '{JSON.stringify({query:`mutation{move(sessionID:"${createdSession.id}",direction:RIGHT){battery score}}`})}'</pre>
					<pre class="bg-gray-50 rounded-lg p-3 text-xs font-mono text-gray-700 mb-2 overflow-x-auto">./statefullgame stdio-mcp</pre>
					<div class="flex gap-3 mt-5">
						<a href="/watch/{createdSession.id}" class="flex-1 bg-[#393c41] text-white text-sm text-center px-4 py-2 rounded-full">Watch</a>
						<button onclick={resetModal} class="flex-1 border border-gray-200 text-sm px-4 py-2 rounded-full hover:bg-gray-50">Close</button>
					</div>
				</div>
			{:else}
				<div class="mb-5">
					<label for="cfg-select" class="block text-sm text-gray-500 mb-2">Configuration</label>
					<select
						id="cfg-select"
						bind:value={selectedConfig}
						class="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-gray-400"
					>
						<option value="">Default</option>
						{#each configs as c}
							<option value={c.configId}>{c.name} — {c.description}</option>
						{/each}
					</select>
				</div>
				{#if createError}
					<p class="text-xs text-red-500 mb-3">{createError}</p>
				{/if}
				<div class="flex gap-3">
					<button
						onclick={createSession}
						disabled={creating}
						class="flex-1 bg-[#393c41] text-white text-sm px-4 py-2 rounded-full disabled:opacity-50"
					>
						{creating ? 'Creating…' : 'Create'}
					</button>
					<button onclick={resetModal} class="flex-1 border border-gray-200 text-sm px-4 py-2 rounded-full hover:bg-gray-50">Cancel</button>
				</div>
			{/if}
		</div>
	</div>
{/if}
