<script lang="ts">
	import { page } from '$app/stores';
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { createClient as createWsClient } from 'graphql-ws';
	import CaveMode from '$lib/CaveMode.svelte';

	const GAME_STATE_QUERY = `
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName
				playerPos { x y }
				grid { type visited id }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const SESSION_SUBSCRIPTION = `
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName
				playerPos { x y }
				grid { type visited id }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const client = getContextClient();
	const sessionId = $page.params.id ?? '';

	const SESSION_QUERY = `
		query Session($id: ID!) {
			session(id: $id) { id displayName mapName }
		}
	`;

	const sessionQuery = queryStore({ client, query: gql(SESSION_QUERY), variables: { id: sessionId } });
	const sessionDisplayName = $derived($sessionQuery.data?.session?.displayName ?? null);

	// Initial load via query
	const initialQuery = queryStore({
		client,
		query: gql(GAME_STATE_QUERY),
		variables: { sessionID: sessionId }
	});

	type Position = { x: number; y: number };
	type MoveHistoryEntry = {
		fromPosition: Position;
		toPosition: Position;
		success: boolean;
	};
	type GameState = {
		battery: number;
		maxBattery: number;
		score: number;
		victory: boolean;
		gameOver: boolean;
		totalMoves: number;
		message: string;
		mapName: string;
		playerPos: Position;
		grid: Array<Array<{ type: string; visited: boolean; id: string }>>;
		currentMoves: MoveHistoryEntry[];
	};

	let liveState = $state<GameState | null>(null);
	let messages = $state<string[]>([]);
	let animatedPos = $state<Position | null>(null);
	let animatedTrailKeys = $state<Set<string>>(new Set());
	let isAnimatingMoves = $state(false);
	let lastAnimationSignature = '';

	const gameState = $derived<GameState | null>(liveState ?? $initialQuery.data?.gameState ?? null);
	const displayPlayerPos = $derived(animatedPos ?? gameState?.playerPos ?? null);
	const isLargeMap = $derived((gameState?.grid.length ?? 0) >= 30);

	// Direct graphql-ws subscription — bypasses urql store compatibility issues
	$effect(() => {
		const WS_URL = typeof window !== 'undefined'
			? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/graphql`
			: 'ws://localhost:8080/graphql';

		const wsClient = createWsClient({ url: WS_URL });

		const unsubscribe = wsClient.subscribe(
			{ query: SESSION_SUBSCRIPTION, variables: { sessionID: sessionId } },
			{
				next(data: { data?: { sessionUpdated?: GameState } }) {
					const gs = data.data?.sessionUpdated;
					if (!gs) return;
					liveState = gs;
					if (gs.message) messages = [gs.message, ...messages].slice(0, 10);
				},
				error(err) { console.error('WS error', err); },
				complete() {}
			}
		);

		return () => unsubscribe();
	});

	let caveEnabled = $state(false);
	let caveRadius = $state(3);

	let promptCopied = $state(false);

	const llmPrompt = $derived(`Use this GraphQL API to control an existing Tesla Road Trip game session.

Goal: visit all parks without getting stranded or hitting a building.

Session ID: ${sessionId}
GraphQL endpoint: ${typeof window !== 'undefined' ? window.location.origin : ''}/graphql
Playground: ${typeof window !== 'undefined' ? window.location.origin : ''}/playground
MCP endpoint: https://tesla.ngrok.pro/mcp (Streamable HTTP transport)

To use MCP in Claude Code, configure:
{
  "mcpServers": {
    "tesla-game": {
      "type": "http",
      "url": "https://tesla.ngrok.pro/mcp"
    }
  }
}

GraphQL introspection is enabled. Use the Playground Docs panel or query __schema/__type to discover fields before constructing operations.

## Inspect the API
query {
  __type(name: "GameState") {
    fields { name type { kind name ofType { kind name } } }
  }
}

## Read current session state
query {
  gameState(sessionID: "${sessionId}") {
    mapName
    playerPos { x y }
    battery
    maxBattery
    score
    victory
    gameOver
    message
    localView3x3
    grid { type visited id }
    visitedParks { id visited }
  }
}

## Send one move
mutation {
  move(sessionID: "${sessionId}", direction: RIGHT) {
    success
    message
    attemptedTo { x y tileChar tileType passable }
    gameState { playerPos { x y } battery score victory gameOver }
  }
}

## Send a move sequence
mutation {
  bulkMove(sessionID: "${sessionId}", moves: [UP, RIGHT, DOWN]) {
    success
    movesExecuted
    requestedMoves
    stoppedReason
    stopReasonCode
    truncated
    limit
    gameState { playerPos { x y } battery score victory gameOver }
  }
}

bulkMove accepts at most 50 moves per call. Check success, stoppedReason, stopReasonCode, truncated, gameOver, and victory before sending another operation.

## Manage this session
mutation { reset(sessionID: "${sessionId}") { playerPos { x y } battery score victory gameOver } }
query { history(sessionID: "${sessionId}", page: 1, limit: 20, order: DESC) { totalMoves moves { moveNumber action success battery } } }
mutation { deleteSession(id: "${sessionId}") { message } }

Directions: UP DOWN LEFT RIGHT. Grid coordinates are grid[y][x].`);

	function copyPrompt() {
		navigator.clipboard.writeText(llmPrompt);
		promptCopied = true;
		setTimeout(() => promptCopied = false, 2000);
	}

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

	$effect(() => {
		const moves = gameState?.currentMoves?.filter((move) => move.success) ?? [];
		if (!gameState || moves.length === 0) {
			animatedPos = null;
			animatedTrailKeys = new Set();
			isAnimatingMoves = false;
			lastAnimationSignature = '';
			return;
		}

		const signature = moves
			.map((move) => `${move.fromPosition.x},${move.fromPosition.y}>${move.toPosition.x},${move.toPosition.y}`)
			.join('|');
		if (signature === lastAnimationSignature) return;
		lastAnimationSignature = signature;

		let cancelled = false;
		let index = 0;
		let trail = new Set<string>();
		isAnimatingMoves = true;
		animatedPos = moves[0].fromPosition;
		trail.add(`${moves[0].fromPosition.x},${moves[0].fromPosition.y}`);
		animatedTrailKeys = new Set(trail);

		const step = () => {
			if (cancelled) return;
			const move = moves[index];
			if (!move) {
				animatedPos = gameState.playerPos;
				isAnimatingMoves = false;
				return;
			}

			trail.add(`${move.fromPosition.x},${move.fromPosition.y}`);
			trail.add(`${move.toPosition.x},${move.toPosition.y}`);
			animatedTrailKeys = new Set(trail);
			animatedPos = move.toPosition;
			index += 1;
			setTimeout(step, 140);
		};

		const timer = setTimeout(step, 180);
		return () => {
			cancelled = true;
			clearTimeout(timer);
		};
	});

	const trailKeys = $derived.by(() => {
		const keys = new Set<string>();
		if (!gameState) return keys;
		if (isAnimatingMoves) return animatedTrailKeys;

		for (const move of gameState.currentMoves ?? []) {
			if (!move.success) continue;
			keys.add(`${move.fromPosition.x},${move.fromPosition.y}`);
			keys.add(`${move.toPosition.x},${move.toPosition.y}`);
		}

		return keys;
	});

	function cellVisible(x: number, y: number): boolean {
		if (!caveEnabled || !displayPlayerPos) return true;
		return Math.max(Math.abs(x - displayPlayerPos.x), Math.abs(y - displayPlayerPos.y)) <= caveRadius;
	}
</script>

<svelte:head>
	<title>{sessionDisplayName ?? sessionId} — Tesla Road Trip</title>
</svelte:head>

<div class="max-w-[1900px] mx-auto px-3 sm:px-4 py-4 lg:py-6">
	<div class={`grid grid-cols-1 gap-4 xl:gap-6 items-start ${isLargeMap ? '' : 'lg:grid-cols-[minmax(0,3fr)_minmax(24rem,2fr)]'}`}>
		<!-- left: compact session controls + board -->
		<section class="min-w-0 bg-white rounded-2xl border border-[#e8e8e8] shadow-sm overflow-visible">
			<div class="p-3 sm:p-4 border-b border-gray-100">
				<div class="flex flex-wrap items-start justify-between gap-3">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2 text-xs uppercase tracking-widest text-gray-400">
							<span>Session</span>
							{#if gameState}
								<span>·</span>
								<span class="normal-case tracking-normal font-mono">{gameState.mapName}</span>
							{/if}
						</div>
						<div class="mt-1 flex items-center gap-2">
							{#if sessionDisplayName}
								<span class="text-lg leading-none text-gray-800 font-medium">{sessionDisplayName}</span>
								<button
									onclick={() => navigator.clipboard.writeText(sessionId)}
									class="font-mono text-sm leading-none text-gray-400 hover:text-blue-600 transition-colors"
									title="Copy session ID"
								>({sessionId})</button>
							{:else}
								<button
									onclick={() => navigator.clipboard.writeText(sessionId)}
									class="font-mono text-lg leading-none text-gray-800 hover:text-blue-600 transition-colors"
									title="Copy session ID"
								>{sessionId}</button>
							{/if}
						</div>
					</div>

					{#if gameState}
						<div class="flex-1 min-w-[16rem] max-w-xl">
							<div class="flex justify-between text-xs text-gray-400 mb-1">
								<span>Battery</span><span>{gameState.battery}/{gameState.maxBattery}</span>
							</div>
							<div class="h-2 bg-gray-100 rounded-full overflow-hidden">
								<div
									class="h-full rounded-full transition-all duration-300 {gameState.battery / gameState.maxBattery > 0.5 ? 'bg-green-400' : gameState.battery / gameState.maxBattery > 0.25 ? 'bg-orange-400' : 'bg-red-400'}"
									style="width: {Math.max(0, (gameState.battery / gameState.maxBattery) * 100)}%"
								></div>
							</div>
						</div>

						<div class="grid grid-cols-3 gap-2 text-center">
							<div class="bg-gray-50 rounded-xl px-4 py-2">
								<div class="text-xl font-light leading-tight">{gameState.score}</div>
								<div class="text-[11px] text-gray-400">Parks</div>
							</div>
							<div class="bg-gray-50 rounded-xl px-4 py-2">
								<div class="text-xl font-light leading-tight">{gameState.totalMoves}</div>
								<div class="text-[11px] text-gray-400">Moves</div>
							</div>
							<div class="bg-gray-50 rounded-xl px-4 py-2">
								<div class="text-lg leading-tight {gameState.victory ? 'text-green-500' : gameState.gameOver ? 'text-red-500' : 'text-gray-300'}">
									{gameState.victory ? '🏆' : gameState.gameOver ? '💥' : '🟢'}
								</div>
								<div class="text-[11px] text-gray-400">{gameState.victory ? 'Won' : gameState.gameOver ? 'Crashed' : 'Active'}</div>
							</div>
						</div>
					{:else}
						<p class="text-sm text-gray-400 font-light">Loading…</p>
					{/if}
				</div>
			</div>

			<div class="p-3 sm:p-4 flex items-start justify-start board-pane">
				{#if gameState?.grid}
					<table class="game-board border-collapse" style={`--grid-size: ${gameState.grid.length}; --board-width: ${isLargeMap ? '92vw' : '60vw'}`}>
						<tbody>
						{#each gameState.grid as row, y}
							<tr>
								{#each row as cell, x}
									{@const visible = cellVisible(x, y)}
									{@const isPlayer = displayPlayerPos && x === displayPlayerPos.x && y === displayPlayerPos.y}
									{@const isTrail = visible && trailKeys.has(`${x},${y}`)}
									<td class="game-cell text-center border transition-colors
										{!visible ? 'bg-slate-800 border-slate-700' : cellColorClass(cell.type)}
										{visible && isTrail && !isPlayer ? 'ring-2 ring-inset ring-sky-300' : ''}
										{visible && cell.visited && !isPlayer ? 'opacity-60' : ''}">
										{#if isPlayer}
											{gameState.victory ? '🚗' : gameState.gameOver ? '💥' : '🚗'}
										{:else if isTrail}
											<span class="text-sky-500 leading-none">•</span>
										{/if}
									</td>
								{/each}
							</tr>
						{/each}
						</tbody>
					</table>
				{:else if $initialQuery.fetching}
					<div class="flex items-center justify-center h-64 text-gray-400">
						<div class="text-center">
							<span class="text-4xl block mb-3">🚗</span>
							<p class="text-sm font-light">Loading <code class="font-mono">{sessionId}</code>…</p>
						</div>
					</div>
				{:else if $initialQuery.error}
					<div class="flex items-center justify-center h-64 text-red-400">
						<p class="text-sm">Session not found: {$initialQuery.error.message}</p>
					</div>
				{:else}
					<div class="flex items-center justify-center h-64 text-gray-400">
						<div class="text-center">
							<span class="text-4xl block mb-3">🚗</span>
							<p class="text-sm font-light">Waiting for moves on <code class="font-mono">{sessionId}</code>…</p>
							<p class="text-xs mt-2">Point an AI at this session to see it play</p>
						</div>
					</div>
				{/if}
			</div>

			<div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs text-gray-400 px-4 pb-3">
				<div class="flex flex-wrap items-center gap-x-4 gap-y-1">
					<span><span class="text-sky-400">•</span> movement trail{isAnimatingMoves ? ' · animating route…' : ''}</span>
					<span class="flex items-center gap-x-3 gap-y-1 flex-wrap">
						<span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-red-500"></span> Home</span>
						<span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-emerald-500"></span> Park</span>
						<span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-yellow-400"></span> Charger</span>
						<span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-slate-700"></span> Blocked</span>
						<span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-blue-400"></span> Water</span>
					</span>
				</div>
				<a href="/lobby" class="hover:text-gray-600 transition-colors">← Back to sessions</a>
			</div>

			{#if gameState}
				<div class="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,24rem)] gap-4 border-t border-gray-100 bg-gray-50/40 p-3 sm:p-4">
					<div class="rounded-xl bg-white border border-gray-100 px-4 py-3">
						<div class="flex items-center justify-between gap-3 mb-2">
							<span class="text-[11px] uppercase tracking-widest text-gray-400">Event log</span>
							<span class="text-[11px] text-gray-300">latest first</span>
						</div>
						<div class="space-y-1 max-h-28 overflow-y-auto">
							{#each messages as msg}
								<p class="text-xs text-gray-600 font-light leading-snug">{msg}</p>
							{:else}
								<p class="text-xs text-gray-400 italic">Waiting for moves…</p>
							{/each}
						</div>
					</div>
					<div>
						<span class="text-[11px] uppercase tracking-widest text-gray-400 px-1">View options</span>
						<CaveMode bind:enabled={caveEnabled} bind:radius={caveRadius} />
						<p class="text-xs text-gray-400 mt-2 px-1">Fog Mode is viewer-only and does not affect the AI.</p>
					</div>
				</div>
			{/if}
		</section>

		<!-- right: LLM prompt -->
		<aside class={`min-w-0 bg-white rounded-2xl border border-[#e8e8e8] p-4 shadow-sm ${isLargeMap ? '' : 'lg:sticky lg:top-4'}`}>
			<div class="flex items-start justify-between gap-3 mb-3">
				<div class="min-w-0">
					<span class="text-xs uppercase tracking-widest text-gray-400">Prompt for LLM</span>
					<div class="mt-2 rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm text-blue-900">
						<strong class="font-medium">Copy this into an AI chat</strong> to control session <code class="font-mono font-semibold">{sessionId}</code>.
					</div>
				</div>
				<button
					onclick={copyPrompt}
					class="text-sm px-4 py-2 rounded-full border transition-colors shrink-0 {promptCopied ? 'bg-green-50 border-green-200 text-green-600' : 'border-blue-300 text-blue-700 hover:bg-blue-50'}"
				>{promptCopied ? 'Copied!' : 'Copy'}</button>
			</div>
			<textarea
				readonly
				value={llmPrompt}
				class="w-full text-sm font-mono text-gray-700 bg-gray-50 rounded-xl p-4 resize-none prompt-pane focus:outline-none leading-relaxed border border-gray-100"
				onclick={(e) => (e.target as HTMLTextAreaElement).select()}
			></textarea>
		</aside>
	</div>
</div>

<style>
	.board-pane {
		min-height: calc(100vh - 18rem);
		overflow: visible;
	}

	.prompt-pane {
		height: calc(100vh - 9rem);
		min-height: 34rem;
	}

	.game-board {
		--cell-size: clamp(
			1.75rem,
			min(
				calc((var(--board-width) - 5rem) / var(--grid-size)),
				calc((100vh - 20rem) / var(--grid-size))
			),
			4.1rem
		);
	}

	.game-cell {
		width: var(--cell-size);
		height: var(--cell-size);
		min-width: var(--cell-size);
		font-size: calc(var(--cell-size) * 0.52);
		line-height: 1;
	}

	@media (max-width: 1023px) {
		.game-board {
			--cell-size: clamp(1.75rem, calc((100vw - 4rem) / var(--grid-size)), 3.25rem);
		}
	}
</style>
