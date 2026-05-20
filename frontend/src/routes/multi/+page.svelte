<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { createClient as createWsClient } from 'graphql-ws';
	import { onMount, untrack } from 'svelte';
	import { SESSIONS_QUERY, CONFIGS_QUERY } from '$lib/queries';

	const GAME_STATE_QUERY = `
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message
				playerPos { x y }
				grid { type visited id }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const SESSION_SUBSCRIPTION = `
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message
				playerPos { x y }
				grid { type visited id }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	type MoveEntry = { fromPosition: { x: number; y: number }; toPosition: { x: number; y: number }; success: boolean };
	type SessionState = {
		id: string;
		battery: number; maxBattery: number; score: number;
		victory: boolean; gameOver: boolean; totalMoves: number; message: string;
		playerPos: { x: number; y: number };
		grid: Array<Array<{ type: string; visited: boolean; id: string }>>;
		currentMoves: MoveEntry[];
	};

	const client = getContextClient();
	const sessionsResult = queryStore({ client, query: gql(SESSIONS_QUERY) });
	const configsResult = queryStore({ client, query: gql(CONFIGS_QUERY) });

	let configs = $state<{ configId: string; name: string }[]>([]);
	let allSessions = $state<{ id: string; configName: string }[]>([]);

	// selected config from URL ?config=
	let selectedConfig = $state($page.url.searchParams.get('config') ?? '');

	const filteredIds = $derived<string[]>(
		selectedConfig
			? allSessions.filter((s) => s.configName === selectedConfig).map((s) => s.id)
			: allSessions.map((s) => s.id)
	);

	let states = $state<Map<string, SessionState>>(new Map());
	let sessionOrder = $state<string[]>([]);

	$effect(() => {
		const next = filteredIds; // reactive read
		untrack(() => {
			const cur = new Set(sessionOrder);
			sessionOrder = [...sessionOrder.filter(id => next.includes(id)), ...next.filter(id => !cur.has(id))];
		});
	});

	const baseGrid = $derived.by(() => {
		for (const id of sessionOrder) {
			const s = states.get(id);
			if (s?.grid) return s.grid;
		}
		return null;
	});

	const playerMap = $derived.by(() => {
		const m = new Map<string, number[]>();
		for (const id of sessionOrder) {
			const s = states.get(id);
			if (!s) continue;
			const key = `${s.playerPos.x},${s.playerPos.y}`;
			const idx = sessionOrder.indexOf(id);
			if (!m.has(key)) m.set(key, []);
			m.get(key)!.push(idx);
		}
		return m;
	});

	// trailMap: "x,y" -> array of session indices that have a trail there
	const trailMap = $derived.by(() => {
		const m = new Map<string, number[]>();
		for (const id of sessionOrder) {
			const s = states.get(id);
			if (!s?.currentMoves) continue;
			const idx = sessionOrder.indexOf(id);
			for (const mv of s.currentMoves) {
				if (!mv.success) continue;
				for (const key of [`${mv.fromPosition.x},${mv.fromPosition.y}`, `${mv.toPosition.x},${mv.toPosition.y}`]) {
					if (!m.has(key)) m.set(key, []);
					if (!m.get(key)!.includes(idx)) m.get(key)!.push(idx);
				}
			}
		}
		return m;
	});

	const carColors = ['🚗','🚕','🚙','🏎️','🚓','🚑','🚒','🛻'];
	const dotColors = ['#3b82f6','#f59e0b','#10b981','#ef4444','#8b5cf6','#f97316','#06b6d4','#ec4899'];

	const wsUnsubs = new Map<string, () => void>();

	async function subscribeSession(id: string) {
		if (wsUnsubs.has(id)) return;
		// mark slot so duplicate calls don't race
		wsUnsubs.set(id, () => {});

		// initial state via fetch
		const graphqlUrl = typeof window !== 'undefined'
			? `${window.location.origin}/graphql`
			: 'http://localhost:8000/graphql';
		try {
			const res = await fetch(graphqlUrl, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ query: GAME_STATE_QUERY, variables: { sessionID: id } })
			});
			const json = await res.json();
			const gs = json?.data?.gameState;
			if (gs) { states.set(id, { id, ...gs }); states = new Map(states); }
		} catch {}

		// live updates via WS
		const WS_URL = typeof window !== 'undefined'
			? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/graphql`
			: 'ws://localhost:8000/graphql';
		const wsc = createWsClient({ url: WS_URL });
		const unsub = wsc.subscribe(
			{ query: SESSION_SUBSCRIPTION, variables: { sessionID: id } },
			{
				next(data: { data?: { sessionUpdated?: Omit<SessionState, 'id'> } }) {
					const gs = data.data?.sessionUpdated;
					if (gs) { states.set(id, { id, ...gs }); states = new Map(states); }
				},
				error() {},
				complete() {}
			}
		);
		wsUnsubs.set(id, unsub);
	}

	function unsubscribeSession(id: string) {
		wsUnsubs.get(id)?.();
		wsUnsubs.delete(id);
		states.delete(id);
		states = new Map(states);
	}

	$effect(() => {
		const ids = new Set(filteredIds);
		for (const id of ids) subscribeSession(id);
		for (const id of wsUnsubs.keys()) {
			if (!ids.has(id)) unsubscribeSession(id);
		}
	});

	function selectConfig(id: string) {
		selectedConfig = id;
		const url = new URL(window.location.href);
		if (id) url.searchParams.set('config', id);
		else url.searchParams.delete('config');
		goto(url.pathname + url.search, { replaceState: true, noScroll: true });
	}

	let pollInterval: ReturnType<typeof setInterval>;
	onMount(() => {
		const u1 = sessionsResult.subscribe((r) => {
			const d = r?.data?.sessions?.sessions;
			if (d) allSessions = d;
		});
		const u2 = configsResult.subscribe((r) => {
			const d = r?.data?.configs;
			if (d) configs = d;
		});
		pollInterval = setInterval(() => {
			sessionsResult.reexecute?.({ requestPolicy: 'network-only' });
		}, 10_000);
		return () => { u1(); u2(); clearInterval(pollInterval); for (const f of wsUnsubs.values()) f(); };
	});

	const cellEmoji: Record<string, string> = {
		home: '🏠', park: '🌳', supercharger: '⚡', water: '💧', building: '🏢'
	};
</script>

<svelte:head>
	<title>Multi-Watch — Tesla Road Trip</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 py-6">
	<div class="flex items-center gap-3 mb-5 flex-wrap">
		<h1 class="text-xl font-light text-[#393c41] mr-2">Multi-Watch</h1>
		<button
			onclick={() => selectConfig('')}
			class="text-xs px-3 py-1 rounded-full border transition-colors {selectedConfig === '' ? 'bg-[#393c41] text-white border-[#393c41]' : 'border-gray-200 text-gray-500 hover:bg-gray-50'}"
		>All <span class="opacity-60">{allSessions.length}</span></button>
		{#each configs as c}
			{@const count = allSessions.filter(s => s.configName === c.configId).length}
			{#if count > 0}
				<button
					onclick={() => selectConfig(c.configId)}
					class="text-xs px-3 py-1 rounded-full border transition-colors {selectedConfig === c.configId ? 'bg-[#393c41] text-white border-[#393c41]' : 'border-gray-200 text-gray-500 hover:bg-gray-50'}"
				>{c.name} <span class="opacity-60">{count}</span></button>
			{/if}
		{/each}
		<a href="/" class="ml-auto text-xs text-gray-400 hover:text-gray-600 transition-colors">← Lobby</a>
	</div>

	<div class="flex flex-col lg:flex-row gap-6">
		<!-- shared grid -->
		<div class="flex-1 min-w-0">
			<div class="bg-white rounded-2xl border border-[#e8e8e8] p-4 shadow-sm overflow-auto">
				{#if baseGrid}
					<table class="border-collapse">
						<tbody>
						{#each baseGrid as row, y}
							<tr>
								{#each row as cell, x}
									{@const players = playerMap.get(`${x},${y}`) ?? []}
									{@const hasPlayer = players.length > 0}
									{@const allOver = hasPlayer && players.every(i => states.get(sessionOrder[i])?.gameOver)}
									{@const trailIdxs = !hasPlayer && cell.type === 'road' ? (trailMap.get(`${x},${y}`) ?? []) : []}
									{@const isTrail = trailIdxs.length > 0}
									<td class="w-9 h-9 text-center text-base border border-gray-50 transition-colors
										{cell.type === 'building' || cell.type === 'water' ? 'bg-gray-100' : 'bg-white'}
										{cell.visited && !hasPlayer && !isTrail ? 'opacity-40' : ''}">
										{#if hasPlayer}
											{#if players.length === 1}
												{states.get(sessionOrder[players[0]])?.gameOver ? '💥' : carColors[players[0] % carColors.length]}
											{:else}
												<span class="flex flex-wrap items-center justify-center gap-0 leading-none text-[10px]">
													{#each players as pidx}
														<span>{states.get(sessionOrder[pidx])?.gameOver ? '💥' : carColors[pidx % carColors.length]}</span>
													{/each}
												</span>
											{/if}
										{:else if isTrail}
											<span class="flex items-center justify-center gap-px w-full h-full">
												{#each trailIdxs as idx}
													<span class="inline-block rounded-full w-1 h-1 shrink-0" style="background:{dotColors[idx % dotColors.length]}"></span>
												{/each}
											</span>
										{:else if cell.type !== 'road'}
											{cellEmoji[cell.type] ?? ''}
										{/if}
									</td>
								{/each}
							</tr>
						{/each}
						</tbody>
					</table>
				{:else if filteredIds.length === 0 && allSessions.length > 0}
					<div class="flex items-center justify-center h-64 text-gray-400">
						<p class="text-sm">No sessions for this config.</p>
					</div>
				{:else}
					<div class="flex items-center justify-center h-64 text-gray-400">
						<span class="text-4xl">🚗</span>
					</div>
				{/if}
			</div>
		</div>

		<!-- session list -->
		<div class="w-full lg:w-56 shrink-0 flex flex-col gap-2">
			{#each sessionOrder as id, i}
				{@const s = states.get(id)}
				<a href="/watch/{id}" class="bg-white rounded-xl border border-[#e8e8e8] px-3 py-2.5 shadow-sm hover:shadow-md transition-shadow flex items-center gap-2">
					<span class="text-lg leading-none">{carColors[i % carColors.length]}</span>
					<div class="flex-1 min-w-0">
						<div class="flex items-center justify-between gap-1">
							<span class="font-mono text-xs font-medium text-[#393c41]">{id}</span>
							{#if s}
								<span class="text-xs shrink-0 {s.victory ? 'text-green-500' : s.gameOver ? 'text-red-500' : 'text-gray-400'}">
									{s.victory ? '🏆' : s.gameOver ? '💥' : '🟢'}
								</span>
							{/if}
						</div>
						{#if s}
							<div class="h-1 bg-gray-100 rounded-full overflow-hidden mt-1">
								<div class="h-full rounded-full {s.battery / s.maxBattery > 0.5 ? 'bg-green-400' : s.battery / s.maxBattery > 0.25 ? 'bg-orange-400' : 'bg-red-400'}"
									style="width:{Math.max(0, s.battery / s.maxBattery * 100)}%"></div>
							</div>
							<div class="flex gap-2 text-xs text-gray-400 mt-0.5">
								<span>🌳 {s.score}</span><span>📍 {s.totalMoves}</span>
							</div>
						{/if}
					</div>
				</a>
			{/each}
			{#if sessionOrder.length === 0}
				<p class="text-xs text-gray-400 text-center py-4">No sessions</p>
			{/if}
		</div>
	</div>
</div>
