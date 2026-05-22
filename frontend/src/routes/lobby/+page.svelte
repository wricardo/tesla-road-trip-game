<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { onMount } from 'svelte';
	import { SESSIONS_QUERY, UPDATE_SESSION_MUTATION } from '$lib/queries';

	const client = getContextClient();
	const sessionsResult = queryStore({ client, query: gql(SESSIONS_QUERY) });

	type SessionCard = {
		id: string;
		displayName: string | null;
		mapName: string;
		battery: number;
		maxBattery: number;
		score: number;
		victory: boolean;
		gameOver: boolean;
		totalMoves: number;
	};

	let sessionMap = $state<Map<string, SessionCard>>(new Map());
	let editingId = $state<string | null>(null);
	let editValue = $state('');

	$effect(() => {
		const data = $sessionsResult?.data?.sessions?.sessions;
		if (!data) return;
		const m = new Map<string, SessionCard>();
		for (const s of data) {
			m.set(s.id, {
				id: s.id,
				displayName: s.displayName ?? null,
				mapName: s.mapName,
				battery: s.gameState.battery,
				maxBattery: s.gameState.maxBattery,
				score: s.gameState.score,
				victory: s.gameState.victory,
				gameOver: s.gameState.gameOver,
				totalMoves: s.gameState.totalMoves
			});
		}
		sessionMap = m;
	});

	function startEdit(s: SessionCard, e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		editingId = s.id;
		editValue = s.displayName ?? '';
	}

	async function commitEdit(id: string, e: Event) {
		e.preventDefault();
		const name = editValue.trim();
		if (name) {
			await client.mutation(gql(UPDATE_SESSION_MUTATION), { id, displayName: name }).toPromise();
			const s = sessionMap.get(id);
			if (s) { s.displayName = name; sessionMap = new Map(sessionMap); }
		}
		editingId = null;
	}

	function cancelEdit() { editingId = null; }

	let pollInterval: ReturnType<typeof setInterval>;
	onMount(() => {
		pollInterval = setInterval(() => {
			sessionsResult.reexecute?.({ requestPolicy: 'network-only' });
		}, 10_000);
		return () => clearInterval(pollInterval);
	});

	const sessions = $derived(Array.from(sessionMap.values()));
</script>

<svelte:head>
	<title>Live Sessions — Tesla Road Trip</title>
</svelte:head>

<div class="bg-white border-b border-[#e8e8e8]">
	<div class="max-w-7xl mx-auto px-6 py-12 lg:py-16">
		<p class="text-xs font-bold uppercase tracking-widest text-red-500 mb-4">● Live agent telemetry</p>
		<div class="flex flex-col lg:flex-row lg:items-end lg:justify-between gap-6">
			<div>
				<h1 class="text-4xl lg:text-5xl font-light text-[#171a20] tracking-tight mb-4">Live Sessions</h1>
				<p class="text-lg text-gray-500 font-light max-w-2xl">
					Watch AI agents drive live, inspect battery usage and routes, or open a session to take control yourself.
				</p>
			</div>
			<div class="flex flex-wrap gap-3">
				<a href="/" class="bg-[#393c41] text-white text-sm px-5 py-3 rounded-full hover:bg-black transition-colors">+ Create</a>
				<a href="/multi" class="border border-gray-200 bg-white text-[#393c41] text-sm px-5 py-3 rounded-full hover:border-gray-400 transition-colors">Multi-watch →</a>
			</div>
		</div>
	</div>
</div>

<div class="max-w-7xl mx-auto px-6 py-10">
	<div class="flex items-center justify-between mb-6">
		<div>
			<h2 class="text-xl font-light text-[#393c41]">Active road trips</h2>
			<p class="text-sm text-gray-400 mt-0.5">Auto-refreshes every 10 seconds</p>
		</div>
		<button
			onclick={() => sessionsResult.reexecute?.({ requestPolicy: 'network-only' })}
			class="text-sm text-gray-400 hover:text-gray-600 transition-colors"
		>
			Refresh
		</button>
	</div>

	{#if $sessionsResult.fetching && sessions.length === 0}
		<div class="flex items-center justify-center py-24 text-gray-400 bg-white rounded-2xl border border-[#e8e8e8]">
			<div class="text-center"><span class="text-4xl mb-4 block">🚗</span><p class="text-lg font-light">Loading sessions…</p></div>
		</div>
	{:else if sessions.length === 0}
		<div class="flex flex-col items-center justify-center py-24 text-gray-400 bg-white rounded-2xl border border-[#e8e8e8]">
			<span class="text-5xl mb-4">🚗</span>
			<p class="text-lg font-light">No active road trips</p>
			<p class="text-sm mt-2">Create a road trip, then watch your AI drive it live.</p>
			<a href="/" class="mt-6 bg-[#393c41] text-white text-sm px-5 py-3 rounded-full hover:bg-black transition-colors">+ Create</a>
		</div>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
			{#each sessions as s (s.id)}
				<a href="/watch/{s.id}" class="block bg-white rounded-2xl shadow-sm border border-[#e8e8e8] p-5 hover:shadow-md transition-shadow">
					<div class="flex items-center justify-between mb-3">
						<div class="flex items-center gap-2 min-w-0">
							{#if editingId === s.id}
								<form onsubmit={(e) => commitEdit(s.id, e)} class="flex items-center gap-1">
									<input
										type="text"
										bind:value={editValue}
										onclick={(e) => e.stopPropagation()}
										onkeydown={(e) => e.key === 'Escape' && cancelEdit()}
										class="font-mono text-sm border-b border-gray-300 focus:border-[#393c41] outline-none px-1 w-32"
										autofocus
									/>
									<button type="submit" onclick={(e) => e.stopPropagation()} class="text-xs text-green-600 hover:text-green-800">✓</button>
									<button type="button" onclick={(e) => { e.stopPropagation(); cancelEdit(); }} class="text-xs text-gray-400 hover:text-gray-600">✕</button>
								</form>
							{:else}
								<span class="font-mono text-sm text-[#393c41] truncate">{s.displayName ?? 'Session ' + s.id}</span>
								<button onclick={(e) => startEdit(s, e)} class="text-gray-300 hover:text-gray-500 transition-colors flex-shrink-0" title="Rename">✎</button>
							{/if}
						</div>
						<span class="text-xs text-gray-400 bg-gray-100 rounded-full px-2 py-0.5 flex-shrink-0">{s.mapName}</span>
					</div>
					<div class="mb-3">
						<div class="flex justify-between text-xs text-gray-400 mb-1"><span>Battery</span><span>{s.battery}/{s.maxBattery}</span></div>
						<div class="h-1.5 bg-gray-100 rounded-full overflow-hidden">
							<div
								class="h-full rounded-full transition-all {s.battery / s.maxBattery > 0.5 ? 'bg-green-400' : s.battery / s.maxBattery > 0.25 ? 'bg-orange-400' : 'bg-red-400'}"
								style="width: {Math.max(0, (s.battery / s.maxBattery) * 100)}%"
							></div>
						</div>
					</div>
					<div class="flex gap-4 text-xs text-gray-500">
						<span>{s.score} parks</span>
						<span>📍 {s.totalMoves} moves</span>
						<span class="ml-auto {s.victory ? 'text-green-500' : s.gameOver ? 'text-red-500' : 'text-gray-400'}">
							{s.victory ? '🏆 Won' : s.gameOver ? '💥 Crashed' : '🟢 Active'}
						</span>
					</div>
					<div class="mt-4 text-right text-xs font-medium text-[#393c41]">Open →</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
