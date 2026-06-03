<script lang="ts">
	import { createClient as createWsClient } from 'graphql-ws';
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { directionGlyph, hasDirections } from '$lib/directional';

	const GAME_STATE_QUERY = `
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName
				playerPos { x y }
				grid { type visited id allowedDirections }
			}
		}
	`;

	const SESSION_SUBSCRIPTION = `
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName
				playerPos { x y }
				grid { type visited id allowedDirections }
			}
		}
	`;

	let { sessionId }: { sessionId: string } = $props();

	const client = getContextClient();
	const initialQuery = queryStore({
		client,
		query: gql(GAME_STATE_QUERY),
		variables: { sessionID: sessionId }
	});

	type GameState = {
		battery: number; maxBattery: number; score: number;
		victory: boolean; gameOver: boolean; totalMoves: number;
		message: string; mapName: string;
		playerPos: { x: number; y: number };
		grid: Array<Array<{ type: string; visited: boolean; id: string; allowedDirections: string[] }>>;
	};

	let liveState = $state<GameState | null>(null);
	const gs = $derived<GameState | null>(liveState ?? $initialQuery.data?.gameState ?? null);

	$effect(() => {
		const WS_URL = typeof window !== 'undefined'
			? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/graphql`
			: 'ws://localhost:8080/graphql';
		const wsClient = createWsClient({ url: WS_URL });
		const unsub = wsClient.subscribe(
			{ query: SESSION_SUBSCRIPTION, variables: { sessionID: sessionId } },
			{
				next(data: { data?: { sessionUpdated?: GameState } }) {
					const s = data.data?.sessionUpdated;
					if (s) liveState = s;
				},
				error() {},
				complete() {}
			}
		);
		return () => unsub();
	});

	function cellColorClass(type: string): string {
		switch (type) {
			case 'home': return 'bg-red-500';
			case 'park': return 'bg-emerald-500';
			case 'supercharger': return 'bg-yellow-400';
			case 'water': return 'bg-blue-400';
			case 'building': return 'bg-slate-700';
			default: return 'bg-white';
		}
	}
</script>

<a href="/watch/{sessionId}" class="block bg-white rounded-2xl border border-[#e8e8e8] shadow-sm hover:shadow-md transition-shadow overflow-hidden">
	<!-- mini grid -->
	<div class="overflow-hidden bg-gray-50 flex items-center justify-center p-2" style="height: 180px">
		{#if gs?.grid}
			{@const cellSize = Math.min(10, Math.floor(160 / gs.grid.length))}
			<table class="border-collapse" style="line-height:1">
				<tbody>
				{#each gs.grid as row, y}
					<tr>
						{#each row as cell, x}
							{@const isPlayer = x === gs.playerPos.x && y === gs.playerPos.y}
							<td style="width:{cellSize}px;height:{cellSize}px;font-size:{cellSize * 0.7}px"
								class="text-center border-0 {cellColorClass(cell.type)} {cell.visited && !isPlayer ? 'opacity-50' : ''}">
								{#if isPlayer}
									{gs.gameOver ? '💥' : '🚗'}
								{:else if hasDirections(cell)}
									<span class="text-orange-500 font-bold leading-none">{directionGlyph(cell.allowedDirections)}</span>
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
				</tbody>
			</table>
		{:else}
			<span class="text-2xl">🚗</span>
		{/if}
	</div>

	<!-- stats bar -->
	<div class="px-3 py-2 border-t border-gray-100">
		<div class="flex items-center justify-between mb-1.5">
			<div class="flex items-center gap-1.5">
				<span class="font-mono text-xs font-medium text-[#393c41]">{sessionId}</span>
				{#if gs}
					<span class="text-gray-300">·</span>
					<span class="text-xs text-gray-400">{gs.mapName}</span>
				{/if}
			</div>
			{#if gs}
				<span class="text-xs {gs.victory ? 'text-green-500' : gs.gameOver ? 'text-red-500' : 'text-gray-400'}">
					{gs.victory ? '🏆 Won' : gs.gameOver ? '💥 Crashed' : '🟢 Active'}
				</span>
			{/if}
		</div>
		{#if gs}
			<div class="h-1 bg-gray-100 rounded-full overflow-hidden">
				<div
					class="h-full rounded-full transition-all duration-300 {gs.battery / gs.maxBattery > 0.5 ? 'bg-green-400' : gs.battery / gs.maxBattery > 0.25 ? 'bg-orange-400' : 'bg-red-400'}"
					style="width:{Math.max(0, gs.battery / gs.maxBattery * 100)}%"
				></div>
			</div>
			<div class="flex gap-3 text-xs text-gray-400 mt-1">
				<span>Parks {gs.score}</span>
				<span>📍 {gs.totalMoves}</span>
			</div>
		{/if}
	</div>
</a>
