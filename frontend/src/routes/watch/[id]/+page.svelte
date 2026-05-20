<script lang="ts">
	import { page } from '$app/stores';
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { createClient as createWsClient } from 'graphql-ws';
	import CaveMode from '$lib/CaveMode.svelte';

	const GAME_STATE_QUERY = `
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message configName
				playerPos { x y }
				grid { type visited id }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const SESSION_SUBSCRIPTION = `
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message configName
				playerPos { x y }
				grid { type visited id }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const client = getContextClient();
	const sessionId = $page.params.id ?? '';

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
		configName: string;
		playerPos: Position;
		grid: Array<Array<{ type: string; visited: boolean; id: string }>>;
		currentMoves: MoveHistoryEntry[];
	};

	let liveState = $state<GameState | null>(null);
	let messages = $state<string[]>([]);

	const gameState = $derived<GameState | null>(liveState ?? $initialQuery.data?.gameState ?? null);

	// Direct graphql-ws subscription — bypasses urql store compatibility issues
	$effect(() => {
		const WS_URL = typeof window !== 'undefined'
			? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/graphql`
			: 'ws://localhost:8000/graphql';

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

	const llmPrompt = $derived(`You are playing Tesla Road Trip — a grid-based navigation game.

Session ID: ${sessionId}
GraphQL endpoint: ${typeof window !== 'undefined' ? window.location.origin : ''}/graphql

Goal: collect all parks (🌳). Charging tiles (🏠🏠 home, ⚡ supercharger) restore battery to max. Buildings (🏢) and water (💧) are impassable. Each move costs 1 battery.

## Read state
query { gameState(sessionID: "${sessionId}") { playerPos { x y } battery maxBattery score victory gameOver message localView3x3 grid { type visited id } visitedParks { id visited } } }

## Single move
mutation { move(sessionID: "${sessionId}", direction: RIGHT) { success message gameState { playerPos { x y } battery victory gameOver } } }

## Execute long route — reset + multiple bulkMove aliases in one request (saves round trips)
mutation {
  reset(sessionID: "${sessionId}") { battery score }
  c1: bulkMove(sessionID: "${sessionId}", moves: [UP,UP,RIGHT,RIGHT,DOWN]) { movesExecuted success stoppedReason gameState { playerPos { x y } battery victory gameOver } }
  c2: bulkMove(sessionID: "${sessionId}", moves: [LEFT,LEFT,UP,UP,RIGHT]) { movesExecuted success stoppedReason gameState { playerPos { x y } battery victory gameOver } }
}

Use aliases (c1:, c2:, c3: …) to chain up to ~50 moves per bulkMove in a single mutation. Each bulkMove resumes from where the previous left off. Check stoppedReason after each chunk — if "wall" or "battery", replan.

Grid is grid[y][x]. Directions: UP DOWN LEFT RIGHT.`);

	function copyPrompt() {
		navigator.clipboard.writeText(llmPrompt);
		promptCopied = true;
		setTimeout(() => promptCopied = false, 2000);
	}

	const cellEmoji: Record<string, string> = {
		home: '🏠', park: '🌳', supercharger: '⚡', water: '💧', building: '🏢'
	};

	const trailKeys = $derived.by(() => {
		const keys = new Set<string>();
		if (!gameState) return keys;

		for (const move of gameState.currentMoves ?? []) {
			if (!move.success) continue;
			keys.add(`${move.fromPosition.x},${move.fromPosition.y}`);
			keys.add(`${move.toPosition.x},${move.toPosition.y}`);
		}

		return keys;
	});

	function cellVisible(x: number, y: number): boolean {
		if (!caveEnabled || !gameState) return true;
		return Math.max(Math.abs(x - gameState.playerPos.x), Math.abs(y - gameState.playerPos.y)) <= caveRadius;
	}
</script>

<svelte:head>
	<title>{sessionId} — Tesla Road Trip</title>
</svelte:head>

<div class="max-w-[1800px] mx-auto px-4 py-8 flex flex-col lg:flex-row gap-6 xl:gap-8">
	<!-- left: grid -->
	<div class="flex-1 min-w-0">
		<div class="bg-white rounded-2xl border border-[#e8e8e8] p-4 sm:p-6 shadow-sm overflow-auto flex items-center justify-center min-h-[55vh]">
			{#if gameState?.grid}
				<table class="game-board border-collapse" style={`--grid-size: ${gameState.grid.length}`}>
					<tbody>
					{#each gameState.grid as row, y}
						<tr>
							{#each row as cell, x}
								{@const visible = cellVisible(x, y)}
								{@const isPlayer = x === gameState.playerPos.x && y === gameState.playerPos.y}
								{@const isTrail = visible && trailKeys.has(`${x},${y}`)}
								<td class="game-cell text-center border transition-colors
									{!visible ? 'bg-gray-900 border-gray-900' : cell.type === 'building' || cell.type === 'water' ? 'bg-gray-100 border-gray-50' : isTrail && !isPlayer ? 'bg-sky-50 border-sky-100' : 'bg-white border-gray-50'}
									{cell.visited && !isPlayer ? 'opacity-60' : ''}">
									{#if isPlayer}
										{gameState.gameOver ? '💥' : '🚗'}
									{:else if visible && cell.type !== 'road'}
										{cellEmoji[cell.type] ?? ''}
									{:else if isTrail}
										<span class="text-sky-400 leading-none">•</span>
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

		<div class="flex items-center gap-x-4 text-xs text-gray-400 mt-2 px-1">
			<span><span class="text-sky-400">•</span> movement trail</span>
		</div>

		<!-- LLM prompt block -->
		<div class="bg-white rounded-2xl border border-[#e8e8e8] p-4 shadow-sm mt-4">
			<div class="flex items-center justify-between mb-2">
				<span class="text-xs uppercase tracking-widest text-gray-400">Prompt for LLM</span>
				<button
					onclick={copyPrompt}
					class="text-xs px-3 py-1 rounded-full border transition-colors {promptCopied ? 'bg-green-50 border-green-200 text-green-600' : 'border-gray-200 text-gray-500 hover:bg-gray-50'}"
				>{promptCopied ? 'Copied!' : 'Copy'}</button>
			</div>
			<textarea
				readonly
				value={llmPrompt}
				class="w-full text-xs font-mono text-gray-600 bg-gray-50 rounded-lg p-2 resize-none h-28 focus:outline-none leading-relaxed"
				onclick={(e) => (e.target as HTMLTextAreaElement).select()}
			></textarea>
		</div>
	</div>

	<!-- right: stats panel -->
	<div class="w-full lg:w-80 flex flex-col gap-4 shrink-0">
		<div class="bg-white rounded-2xl border border-[#e8e8e8] p-5 shadow-sm">
			<div class="flex items-center justify-between mb-5">
				<div class="flex items-center gap-2">
					<span class="text-xs uppercase tracking-widest text-gray-400">Session</span>
					{#if gameState}
						<span class="text-xs text-gray-400">·</span>
						<span class="text-xs text-gray-400 font-mono">{gameState.configName}</span>
					{/if}
				</div>
				<button
					onclick={() => navigator.clipboard.writeText(sessionId)}
					class="font-mono text-sm bg-gray-50 px-2 py-0.5 rounded hover:bg-gray-100 transition-colors"
					title="Copy session ID"
				>{sessionId}</button>
			</div>

			{#if gameState}
				<div class="mb-5">
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

				<div class="grid grid-cols-3 gap-3 mb-5">
					<div class="bg-gray-50 rounded-xl p-3 text-center">
						<div class="text-2xl font-light">{gameState.score}</div>
						<div class="text-xs text-gray-400 mt-0.5">Parks</div>
					</div>
					<div class="bg-gray-50 rounded-xl p-3 text-center">
						<div class="text-2xl font-light">{gameState.totalMoves}</div>
						<div class="text-xs text-gray-400 mt-0.5">Moves</div>
					</div>
					<div class="bg-gray-50 rounded-xl p-3 text-center">
						<div class="text-xl {gameState.victory ? 'text-green-500' : gameState.gameOver ? 'text-red-500' : 'text-gray-300'}">
							{gameState.victory ? '🏆' : gameState.gameOver ? '💥' : '🟢'}
						</div>
						<div class="text-xs text-gray-400 mt-0.5">
							{gameState.victory ? 'Won' : gameState.gameOver ? 'Crashed' : 'Active'}
						</div>
					</div>
				</div>

				<div class="border-t border-gray-100 pt-4">
					<span class="text-xs uppercase tracking-widest text-gray-400 block mb-2">Events</span>
					<div class="space-y-1.5 max-h-36 overflow-y-auto">
						{#each messages as msg}
							<p class="text-xs text-gray-600 font-light leading-snug">{msg}</p>
						{:else}
							<p class="text-xs text-gray-400 italic">Waiting for moves…</p>
						{/each}
					</div>
				</div>

				<div class="border-t border-gray-100 pt-4 mt-3">
					<CaveMode bind:enabled={caveEnabled} bind:radius={caveRadius} />
					<p class="text-xs text-gray-400 mt-1">Viewer-only fog — does not affect the AI</p>
				</div>
			{:else}
				<p class="text-sm text-gray-400 font-light text-center py-8">Loading…</p>
			{/if}
		</div>

		<a href="/" class="text-xs text-center text-gray-400 hover:text-gray-600 transition-colors mt-1">← Back to lobby</a>
	</div>
</div>

<style>
	.game-board {
		--cell-size: clamp(
			2rem,
			min(
				calc((100vw - 30rem - 8rem) / var(--grid-size)),
				calc((100vh - 16rem) / var(--grid-size))
			),
			4.5rem
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
