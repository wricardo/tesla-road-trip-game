<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { createClient as createWsClient } from 'graphql-ws';
	import { onMount, untrack } from 'svelte';
	import { SESSIONS_QUERY, MAPS_QUERY } from '$lib/queries';
	import { directionGlyph, hasDirections } from '$lib/directional';

	const GAME_STATE_QUERY = `
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves
				fogEnabled fogRadius
				playerPos { x y }
				grid { type visited id allowedDirections }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const FOG_GAME_STATE_QUERY = `
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves
				fogEnabled fogRadius
				playerPos { x y }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const SESSION_SUBSCRIPTION = `
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves
				fogEnabled fogRadius
				playerPos { x y }
				grid { type visited id allowedDirections }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const FOG_SESSION_SUBSCRIPTION = `
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves
				fogEnabled fogRadius
				playerPos { x y }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	type Position = { x: number; y: number };
	type MoveEntry = { fromPosition: Position; toPosition: Position; success: boolean };
	type Cell = { type: string; visited: boolean; id: string; allowedDirections: string[] };
	type SessionState = {
		id: string;
		battery: number; maxBattery: number; score: number;
		victory: boolean; gameOver: boolean; totalMoves: number;
		fogEnabled: boolean; fogRadius: number;
		playerPos: { x: number; y: number };
		grid?: Cell[][];
		nearbyGrid?: Cell[][];
		currentMoves: MoveEntry[];
	};

	const client = getContextClient();
	const sessionsResult = queryStore({ client, query: gql(SESSIONS_QUERY) });
	const mapsResult = queryStore({ client, query: gql(MAPS_QUERY) });

	let maps = $state<{ mapId: string; name: string; gridSize: number }[]>([]);
	let allSessions = $state<{ id: string; mapName: string; gameState?: { fogEnabled?: boolean; fogRadius?: number } }[]>([]);

	// selected map from URL ?map=
	const selectedMap = $derived($page.url.searchParams.get('map') ?? '');

	function parseSessionIds(raw: string): string[] {
		return raw.split(',').map((id) => id.trim()).filter(Boolean);
	}

	const availableIds = $derived<string[]>(
		selectedMap
			? allSessions.filter((s) => s.mapName === selectedMap).map((s) => s.id)
			: []
	);
	const sessionMetaByID = $derived.by(() => {
		const m = new Map<string, { fogEnabled: boolean; fogRadius: number }>();
		for (const s of allSessions) {
			m.set(s.id, {
				fogEnabled: !!s.gameState?.fogEnabled,
				fogRadius: s.gameState?.fogRadius ?? 1
			});
		}
		return m;
	});

	const selectedIds = $derived<string[]>(
		selectedMap
			? parseSessionIds($page.url.searchParams.get('sessions') ?? '').filter((id) => availableIds.includes(id))
			: []
	);

	let states = $state<Map<string, SessionState>>(new Map());
	let sessionOrder = $state<string[]>([]);
	let animatedPositions = $state<Map<string, Position>>(new Map());
	let animatedTrails = $state<Map<string, Set<string>>>(new Map());
	let animatingIds = $state<Set<string>>(new Set());
	const animationSignatures = new Map<string, string>();
	const animationMoveCounts = new Map<string, number>();
	const animationTimers = new Map<string, ReturnType<typeof setTimeout>>();

	$effect(() => {
		const next = selectedIds; // reactive read
		untrack(() => {
			const cur = new Set(sessionOrder);
			sessionOrder = [...next.filter((id) => cur.has(id)), ...next.filter((id) => !cur.has(id))];
		});
	});

	const selectedMapGridSize = $derived(maps.find((m) => m.mapId === selectedMap)?.gridSize ?? 0);

	function projectNearbyCell(state: SessionState, x: number, y: number): Cell | null {
		const nearby = state.nearbyGrid;
		if (!nearby?.length) return null;
		const radius = state.fogRadius > 0 ? state.fogRadius : 1;
		const minX = state.playerPos.x - radius;
		const minY = state.playerPos.y - radius;
		const ix = x - minX;
		const iy = y - minY;
		if (iy < 0 || iy >= nearby.length) return null;
		if (ix < 0 || ix >= (nearby[iy]?.length ?? 0)) return null;
		return nearby[iy]?.[ix] ?? null;
	}

	const baseGrid = $derived.by(() => {
		for (const id of sessionOrder) {
			const s = states.get(id);
			if (s?.grid?.length) return s.grid;
		}
		const size = selectedMapGridSize;
		if (size <= 0) return null;
		const rows: Array<Array<Cell | null>> = [];
		for (let y = 0; y < size; y++) {
			const row: Array<Cell | null> = [];
			for (let x = 0; x < size; x++) {
				let cell: Cell | null = null;
				for (const id of sessionOrder) {
					const state = states.get(id);
					if (!state) continue;
					cell = projectNearbyCell(state, x, y);
					if (cell) break;
				}
				row.push(cell);
			}
			rows.push(row);
		}
		return rows;
	});

	function setAnimatedPosition(id: string, pos: Position | null) {
		const next = new Map(animatedPositions);
		if (pos) next.set(id, pos);
		else next.delete(id);
		animatedPositions = next;
	}

	function setAnimatedTrail(id: string, trail: Set<string> | null) {
		const next = new Map(animatedTrails);
		if (trail) next.set(id, trail);
		else next.delete(id);
		animatedTrails = next;
	}

	function setAnimating(id: string, value: boolean) {
		const next = new Set(animatingIds);
		if (value) next.add(id);
		else next.delete(id);
		animatingIds = next;
	}

	function stopAnimation(id: string) {
		const timer = animationTimers.get(id);
		if (timer) clearTimeout(timer);
		animationTimers.delete(id);
		setAnimating(id, false);
		setAnimatedPosition(id, null);
		setAnimatedTrail(id, null);
	}

	function animateSession(id: string, state: SessionState) {
		const moves = state.currentMoves?.filter((move) => move.success) ?? [];
		if (moves.length === 0) {
			stopAnimation(id);
			animationSignatures.delete(id);
			animationMoveCounts.delete(id);
			return;
		}

		const signature = moves
			.map((move) => `${move.fromPosition.x},${move.fromPosition.y}>${move.toPosition.x},${move.toPosition.y}`)
			.join('|');
		const previousSignature = animationSignatures.get(id);
		const previousMoveCount = animationMoveCounts.get(id) ?? 0;
		if (previousSignature === signature) return;

		animationSignatures.set(id, signature);
		animationMoveCounts.set(id, moves.length);
		if (!previousSignature) {
			// First snapshot for a session may already include long history; render at current position.
			stopAnimation(id);
			return;
		}

		const isAppendOnly = signature.startsWith(`${previousSignature}|`);
		if (!isAppendOnly) {
			// History diverged (reset/rewind/reconnect). Avoid replaying the full path.
			stopAnimation(id);
			return;
		}

		const deltaMoves = moves.slice(previousMoveCount);
		if (deltaMoves.length === 0) return;
		const historicalMoves = moves.slice(0, previousMoveCount);

		stopAnimation(id);
		let index = 0;
		const trail = new Set<string>();
		for (const move of historicalMoves) {
			trail.add(`${move.fromPosition.x},${move.fromPosition.y}`);
			trail.add(`${move.toPosition.x},${move.toPosition.y}`);
		}
		setAnimating(id, true);
		setAnimatedPosition(id, deltaMoves[0].fromPosition);
		trail.add(`${deltaMoves[0].fromPosition.x},${deltaMoves[0].fromPosition.y}`);
		setAnimatedTrail(id, new Set(trail));

		const step = () => {
			const latest = states.get(id);
			const move = deltaMoves[index];
			if (!latest || !move) {
				setAnimatedPosition(id, latest?.playerPos ?? state.playerPos);
				setAnimating(id, false);
				animationTimers.delete(id);
				return;
			}

			trail.add(`${move.fromPosition.x},${move.fromPosition.y}`);
			trail.add(`${move.toPosition.x},${move.toPosition.y}`);
			setAnimatedTrail(id, new Set(trail));
			setAnimatedPosition(id, move.toPosition);
			index += 1;
			animationTimers.set(id, setTimeout(step, 140));
		};

		animationTimers.set(id, setTimeout(step, 180));
	}

	function setSessionState(id: string, nextState: SessionState) {
		states.set(id, nextState);
		states = new Map(states);
		animateSession(id, nextState);
	}

	const playerMap = $derived.by(() => {
		const m = new Map<string, number[]>();
		for (const id of sessionOrder) {
			const s = states.get(id);
			if (!s) continue;
			const pos = animatedPositions.get(id) ?? s.playerPos;
			const key = `${pos.x},${pos.y}`;
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
			const animatedTrail = animatingIds.has(id) ? animatedTrails.get(id) : null;

			if (animatedTrail) {
				for (const key of animatedTrail) {
					if (!m.has(key)) m.set(key, []);
					if (!m.get(key)!.includes(idx)) m.get(key)!.push(idx);
				}
				continue;
			}

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
		const meta = sessionMetaByID.get(id);
		const isFogSession = !!meta?.fogEnabled;
		const graphqlUrl = typeof window !== 'undefined'
			? `${window.location.origin}/graphql`
			: 'http://localhost:8080/graphql';
		try {
			const res = await fetch(graphqlUrl, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ query: isFogSession ? FOG_GAME_STATE_QUERY : GAME_STATE_QUERY, variables: { sessionID: id } })
			});
			const json = await res.json();
			const gs = json?.data?.gameState;
			if (gs) setSessionState(id, { id, ...gs });
		} catch {}

		// live updates via WS
		const WS_URL = typeof window !== 'undefined'
			? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/graphql`
			: 'ws://localhost:8080/graphql';
		const wsc = createWsClient({ url: WS_URL });
		const unsub = wsc.subscribe(
			{ query: isFogSession ? FOG_SESSION_SUBSCRIPTION : SESSION_SUBSCRIPTION, variables: { sessionID: id } },
			{
				next(data: { data?: { sessionUpdated?: Omit<SessionState, 'id'> } }) {
					const gs = data.data?.sessionUpdated;
					if (gs) setSessionState(id, { id, ...gs });
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
		stopAnimation(id);
		animationSignatures.delete(id);
		animationMoveCounts.delete(id);
		states.delete(id);
		states = new Map(states);
	}

	$effect(() => {
		const ids = new Set(selectedIds);
		for (const id of ids) subscribeSession(id);
		for (const id of wsUnsubs.keys()) {
			if (!ids.has(id)) unsubscribeSession(id);
		}
	});

	function selectMap(id: string) {
		const url = new URL(window.location.href);
		if (id) url.searchParams.set('map', id);
		else url.searchParams.delete('map');
		url.searchParams.delete('sessions');
		goto(url.pathname + url.search, { replaceState: true, noScroll: true });
	}

	function toggleSession(id: string) {
		if (!selectedMap) return;
		const url = new URL(window.location.href);
		const next = parseSessionIds(url.searchParams.get('sessions') ?? '');
		const idx = next.indexOf(id);
		if (idx >= 0) next.splice(idx, 1);
		else next.push(id);
		if (next.length > 0) url.searchParams.set('sessions', next.join(','));
		else url.searchParams.delete('sessions');
		goto(url.pathname + url.search, { replaceState: true, noScroll: true });
	}

	let pollInterval: ReturnType<typeof setInterval>;
	onMount(() => {
		const u1 = sessionsResult.subscribe((r) => {
			const d = r?.data?.sessions?.sessions;
			if (d) allSessions = d;
		});
		const u2 = mapsResult.subscribe((r) => {
			const d = r?.data?.maps;
			if (d) maps = d;
		});
		pollInterval = setInterval(() => {
			sessionsResult.reexecute?.({ requestPolicy: 'network-only' });
		}, 10_000);
		return () => {
			u1();
			u2();
			clearInterval(pollInterval);
			for (const f of wsUnsubs.values()) f();
			for (const timer of animationTimers.values()) clearTimeout(timer);
		};
	});

	function cellColorClass(type: string): string {
		switch (type) {
			case 'home': return 'bg-red-500 border-red-200';
			case 'park': return 'bg-emerald-500 border-emerald-200';
			case 'supercharger': return 'bg-yellow-400 border-yellow-200';
			case 'water': return 'bg-blue-400 border-blue-200';
			case 'building': return 'bg-slate-700 border-slate-600';
			default: return 'bg-white border-gray-50';
		}
	}
</script>

<svelte:head>
	<title>Multi-Watch — Tesla Road Trip</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 py-6">
	<div class="flex items-center gap-3 mb-5 flex-wrap">
		<h1 class="text-xl font-light text-[#393c41] mr-2">Multi-Watch</h1>
		{#each maps as m}
			{@const count = allSessions.filter(s => s.mapName === m.mapId).length}
			{#if count > 0}
				<button
					onclick={() => selectMap(m.mapId)}
					class="text-xs px-3 py-1 rounded-full border transition-colors {selectedMap === m.mapId ? 'bg-[#393c41] text-white border-[#393c41]' : 'border-gray-200 text-gray-500 hover:bg-gray-50'}"
				>{m.name} <span class="opacity-60">{count}</span></button>
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
									{@const trailIdxs = !hasPlayer ? (trailMap.get(`${x},${y}`) ?? []) : []}
									{@const isTrail = trailIdxs.length > 0}
									<td class="w-9 h-9 text-center text-base border transition-colors
										{cell ? cellColorClass(cell.type) : 'bg-slate-100 border-slate-200'}
										{isTrail && !hasPlayer ? 'ring-2 ring-inset ring-sky-300' : ''}
										{cell?.visited && !hasPlayer && !isTrail ? 'opacity-40' : ''}">
										{#if hasPlayer}
											{#if players.length === 1}
												{!states.get(sessionOrder[players[0]])?.victory && states.get(sessionOrder[players[0]])?.gameOver ? '💥' : carColors[players[0] % carColors.length]}
											{:else}
												<span class="flex flex-wrap items-center justify-center gap-0 leading-none text-[10px]">
													{#each players as pidx}
														<span>{!states.get(sessionOrder[pidx])?.victory && states.get(sessionOrder[pidx])?.gameOver ? '💥' : carColors[pidx % carColors.length]}</span>
													{/each}
												</span>
											{/if}
										{:else if isTrail}
											<span class="flex items-center justify-center gap-px w-full h-full">
												{#each trailIdxs as idx}
													<span class="inline-block rounded-full w-1 h-1 shrink-0" style="background:{dotColors[idx % dotColors.length]}"></span>
												{/each}
											</span>
										{:else if cell && hasDirections(cell)}
											<span class="text-orange-500 font-bold leading-none">{directionGlyph(cell.allowedDirections)}</span>
										{/if}
									</td>
								{/each}
							</tr>
						{/each}
						</tbody>
					</table>
				{:else if !selectedMap}
					<div class="flex flex-col items-center justify-center h-64 text-gray-400 gap-3">
						<span class="text-4xl">🗺️</span>
						<p class="text-sm">Select a map above to choose sessions.</p>
					</div>
				{:else if availableIds.length === 0}
					<div class="flex items-center justify-center h-64 text-gray-400">
						<p class="text-sm">No sessions for this map.</p>
					</div>
				{:else if selectedIds.length === 0}
					<div class="flex flex-col items-center justify-center h-64 text-gray-400 gap-3">
						<span class="text-4xl">☑️</span>
						<p class="text-sm">Select one or more sessions from the list.</p>
					</div>
				{:else}
					<div class="flex items-center justify-center h-64 text-gray-400">
						<span class="text-4xl">🚗</span>
					</div>
				{/if}
			</div>
		</div>

		<!-- session list -->
		<div class="w-full lg:w-72 shrink-0 flex flex-col gap-2">
			<div class="flex items-center justify-between gap-2 px-1 pb-1">
				<span class="text-[11px] uppercase tracking-widest text-gray-400">Sessions to watch</span>
				<span class="text-[11px] text-gray-300">{selectedIds.length} selected</span>
			</div>
			{#each availableIds as id, i}
				{@const s = states.get(id)}
				{@const meta = sessionMetaByID.get(id)}
				<label class="bg-white rounded-xl border border-[#e8e8e8] px-3 py-2.5 shadow-sm hover:shadow-md transition-shadow flex items-start gap-2 cursor-pointer">
					<input
						type="checkbox"
						class="mt-1 accent-[#393c41]"
						checked={selectedIds.includes(id)}
						onchange={() => toggleSession(id)}
					/>
					<span class="text-lg leading-none mt-0.5">{carColors[i % carColors.length]}</span>
					<div class="flex-1 min-w-0">
						<div class="flex items-center justify-between gap-1">
							<div class="min-w-0">
								<div class="font-mono text-xs font-medium text-[#393c41] truncate">{id}</div>
								<div class="text-[11px] text-gray-400">{s ? `${s.score} parks · ${s.totalMoves} moves` : 'Loading…'}</div>
								{#if meta?.fogEnabled}
									<div class="mt-1 inline-flex items-center rounded-full bg-blue-100 text-blue-700 px-2 py-0.5 text-[10px]">🌫 Fog r{meta.fogRadius}</div>
								{/if}
							</div>
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
						{/if}
					</div>
				</label>
			{/each}
			{#if availableIds.length === 0}
				<p class="text-xs text-gray-400 text-center py-4">No sessions on this map</p>
			{/if}
		</div>
	</div>
</div>
