<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { onMount } from 'svelte';
	import { MAPS_QUERY, SESSIONS_QUERY, UPDATE_SESSION_MUTATION } from '$lib/queries';
	import { mapLabel as labelFor } from '$lib/maps';
	import { filterSessions, sessionStatus, statusCounts, type SessionSort, type StatusFilter } from '$lib/sessions';
	import { absoluteTime, relativeTime } from '$lib/time';

	const client = getContextClient();
	const sessionsResult = queryStore({ client, query: gql(SESSIONS_QUERY) });
	const mapsResult = queryStore({ client, query: gql(MAPS_QUERY) });
	const mapLabel = (mapId: string) => labelFor(mapId, $mapsResult?.data?.maps ?? []);

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
		resetCount: number;
		lastActionAt: string;
		fogEnabled: boolean;
		fogRadius: number;
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
				totalMoves: s.gameState.totalMoves,
				resetCount: s.gameState.resetCount,
				lastActionAt: s.lastActionAt,
				fogEnabled: s.gameState.fogEnabled,
				fogRadius: s.gameState.fogRadius
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
		// Force a fresh fetch whenever this page is entered via client-side navigation.
		sessionsResult.reexecute?.({ requestPolicy: 'network-only' });

		pollInterval = setInterval(() => {
			sessionsResult.reexecute?.({ requestPolicy: 'network-only' });
		}, 10_000);
		return () => clearInterval(pollInterval);
	});

	const sessions = $derived(Array.from(sessionMap.values()));

	let statusFilter = $state<StatusFilter>('all');
	let search = $state('');
	let sortBy = $state<SessionSort>('recent');

	const counts = $derived(statusCounts(sessions));
	const filtered = $derived(filterSessions(sessions, { status: statusFilter, search, sort: sortBy, mapLabel }));

	const filterTabs: { key: StatusFilter; label: string; title?: string }[] = [
		{ key: 'all', label: 'All' },
		{ key: 'playing', label: 'Playing' },
		{ key: 'won', label: 'Won' },
		{ key: 'lost', label: 'Lost', title: 'Crashed or out of battery' }
	];

	// Ticks so relative times ("2m ago") stay current between polls.
	let now = $state(Date.now());
	onMount(() => {
		const t = setInterval(() => (now = Date.now()), 30_000);
		return () => clearInterval(t);
	});

	function lostLabel(s: SessionCard): string {
		return s.battery <= 0 ? '🪫 Out of battery' : '💥 Crashed';
	}
</script>

<svelte:head>
	<title>Sessions — Tesla Road Trip</title>
</svelte:head>

<div class="bg-white border-b border-[#e8e8e8]">
	<div class="max-w-7xl mx-auto px-6 py-12 lg:py-16">
		<p class="text-xs font-bold uppercase tracking-widest text-red-600 mb-4">● Live agent telemetry</p>
		<div class="flex flex-col lg:flex-row lg:items-end lg:justify-between gap-6">
			<div>
				<h1 class="text-4xl lg:text-5xl font-light text-[#171a20] tracking-tight mb-4">Sessions</h1>
				<p class="text-lg text-gray-500 font-light max-w-2xl">
					Watch AI agents drive live, inspect battery usage and routes, or open a session to take control yourself.
				</p>
			</div>
			<div class="flex flex-wrap gap-3">
				<a href="/" class="bg-[#393c41] text-white text-sm px-5 py-3 rounded-full hover:bg-black transition-colors">+ New session</a>
				<a href="/multi" class="border border-gray-200 bg-white text-[#393c41] text-sm px-5 py-3 rounded-full hover:border-gray-400 transition-colors">Watch multiple →</a>
			</div>
		</div>
	</div>
</div>

<div class="max-w-7xl mx-auto px-6 py-10">
	<div class="flex flex-wrap items-end justify-between gap-4 mb-6">
		<div>
			<h2 class="text-xl font-light text-[#393c41]">Road trips</h2>
			<p class="text-sm text-gray-500 mt-0.5">Showing {filtered.length} of {sessions.length} · auto-refreshes every 10 s</p>
		</div>
		<button
			onclick={() => sessionsResult.reexecute?.({ requestPolicy: 'network-only' })}
			class="text-sm border border-gray-200 bg-white rounded-full px-4 py-1.5 text-gray-600 hover:border-gray-400 hover:text-gray-900 transition-colors"
		>
			Refresh
		</button>
	</div>

	<div class="flex flex-wrap items-center gap-3 mb-6">
		<div class="inline-flex rounded-full border border-gray-200 bg-white p-1" role="group" aria-label="Filter by status">
			{#each filterTabs as tab}
				<button
					type="button"
					onclick={() => (statusFilter = tab.key)}
					aria-pressed={statusFilter === tab.key}
					title={tab.title}
					class="text-sm px-4 py-1.5 rounded-full transition-colors {statusFilter === tab.key ? 'bg-[#393c41] text-white' : 'text-gray-600 hover:bg-gray-100'}"
				>{tab.label} <span class="tabular-nums opacity-70">{counts[tab.key]}</span></button>
			{/each}
		</div>
		<input
			type="search"
			bind:value={search}
			placeholder="Search name, id or map"
			aria-label="Search sessions"
			class="flex-1 min-w-[12rem] max-w-sm border border-gray-200 rounded-full px-4 py-2 text-sm bg-white"
		/>
		<label class="flex items-center gap-2 text-sm text-gray-600">
			Sort
			<select bind:value={sortBy} class="border border-gray-200 rounded-full px-3 py-2 text-sm bg-white">
				<option value="recent">Last activity</option>
				<option value="moves">Most moves</option>
				<option value="parks">Most parks</option>
			</select>
		</label>
	</div>

	{#if $sessionsResult.fetching && sessions.length === 0}
		<div class="flex items-center justify-center py-24 text-gray-400 bg-white rounded-2xl border border-[#e8e8e8]">
			<div class="text-center"><span class="text-4xl mb-4 block">🚗</span><p class="text-lg font-light">Loading sessions…</p></div>
		</div>
	{:else if sessions.length === 0}
		<div class="flex flex-col items-center justify-center py-24 text-gray-400 bg-white rounded-2xl border border-[#e8e8e8]">
			<span class="text-5xl mb-4">🚗</span>
			<p class="text-lg font-light">No sessions yet — start one and watch it here live.</p>
			<a href="/" class="mt-6 bg-[#393c41] text-white text-sm px-5 py-3 rounded-full hover:bg-black transition-colors">+ New session</a>
		</div>
	{:else if filtered.length === 0}
		<div class="flex flex-col items-center justify-center py-16 text-gray-500 bg-white rounded-2xl border border-[#e8e8e8]">
			<p class="text-lg font-light">No sessions match</p>
			<button type="button" onclick={() => { statusFilter = 'all'; search = ''; }} class="mt-4 bg-[#393c41] text-white text-sm px-5 py-2.5 rounded-full hover:bg-black transition-colors">Clear filters</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
			{#each filtered as s (s.id)}
				<div class="relative bg-white rounded-2xl shadow-sm border border-[#e8e8e8] p-5 hover:shadow-md focus-within:shadow-md transition-shadow">
					<div class="flex items-center justify-between mb-3">
						<div class="flex items-center gap-2 min-w-0">
							{#if editingId === s.id}
								<form onsubmit={(e) => commitEdit(s.id, e)} class="relative z-10 flex items-center gap-1">
									<input
										type="text"
										bind:value={editValue}
										onclick={(e) => e.stopPropagation()}
										onkeydown={(e) => e.key === 'Escape' && cancelEdit()}
										class="font-mono text-sm border-b border-gray-300 focus:border-[#393c41] outline-none px-1 w-32"
										autofocus
									/>
									<button type="submit" aria-label="Save name" class="text-xs text-green-700 hover:text-green-900">✓</button>
									<button type="button" aria-label="Cancel rename" onclick={cancelEdit} class="text-xs text-gray-500 hover:text-gray-800">✕</button>
								</form>
							{:else}
								<span class="text-sm text-[#393c41] truncate" title={`Session ${s.id}`}>{s.displayName ?? 'Unnamed session'} <span class="font-mono text-xs text-gray-400">{s.id}</span></span>
								<button onclick={(e) => startEdit(s, e)} class="relative z-10 text-gray-500 hover:text-gray-900 transition-colors flex-shrink-0" title="Rename" aria-label="Rename session">✎</button>
							{/if}
						</div>
						<div class="flex items-center gap-2 flex-shrink-0">
							{#if s.fogEnabled}
								<span class="text-xs text-blue-700 bg-blue-100 rounded-full px-2 py-0.5" title={`Fog radius ${s.fogRadius}`}>🌫 Fog r{s.fogRadius}</span>
							{/if}
							<span class="text-xs text-gray-600 bg-gray-100 rounded-full px-2 py-0.5" title={s.mapName}>{mapLabel(s.mapName)}</span>
						</div>
					</div>
					<div class="mb-3">
						<div class="flex justify-between text-xs text-gray-400 mb-1"><span>Battery</span><span>{s.battery} / {s.maxBattery}</span></div>
						<div class="h-1.5 bg-gray-100 rounded-full overflow-hidden">
							<div
								class="h-full rounded-full transition-all {s.battery / s.maxBattery > 0.5 ? 'bg-green-400' : s.battery / s.maxBattery > 0.25 ? 'bg-orange-400' : 'bg-red-400'}"
								style="width: {Math.max(0, (s.battery / s.maxBattery) * 100)}%"
							></div>
						</div>
					</div>
					<div class="flex flex-wrap gap-3 text-xs text-gray-500">
						<span>{s.score} parks</span>
						<span>📍 {s.totalMoves} moves</span>
						<span>↺ {s.resetCount} resets</span>
						<span title={absoluteTime(s.lastActionAt)}>🕒 {relativeTime(s.lastActionAt, now)}</span>
						<span class="ml-auto font-medium {s.victory ? 'text-green-700' : s.gameOver ? 'text-red-700' : 'text-gray-600'}">
							{s.victory ? '🏆 Won' : s.gameOver ? lostLabel(s) : s.totalMoves === 0 ? 'Ready' : '🟢 Playing'}
						</span>
					</div>
					<div class="mt-4 text-right text-xs font-medium text-[#393c41]">
						<a href="/watch/{s.id}" class="hover:underline after:absolute after:inset-0 after:rounded-2xl">{sessionStatus(s) === 'playing' ? 'Play' : 'Watch'} →<span class="sr-only"> {s.displayName ?? s.id}</span></a>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
