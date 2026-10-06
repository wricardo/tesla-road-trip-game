<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { createClient as createWsClient } from 'graphql-ws';
	import { directionGlyph, hasDirections } from '$lib/directional';
	import { MOVE_MUTATION, RESET_MUTATION } from '$lib/queries';

	const GAME_STATE_QUERY = `
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score totalParks message victory gameOver totalMoves resetCount mapName
				fogEnabled fogRadius
				playerPos { x y }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const FULL_GRID_QUERY = `
		query FullGrid($sessionID: ID!, $password: String!) {
			gameState(sessionID: $sessionID) {
				grid(password: $password) { type visited id allowedDirections }
			}
		}
	`;

	const SESSION_SUBSCRIPTION = `
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score totalParks message victory gameOver totalMoves resetCount mapName
				fogEnabled fogRadius
				playerPos { x y }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`;

	const client = getContextClient();
	const sessionId = $page.params.id ?? '';

	const SESSION_QUERY = `
		query Session($id: ID!) {
			session(id: $id) {
				id
				displayName
				mapName
				gameMap { gridSize }
			}
		}
	`;

	const sessionQuery = queryStore({ client, query: gql(SESSION_QUERY), variables: { id: sessionId } });
	const sessionDisplayName = $derived($sessionQuery.data?.session?.displayName ?? null);
	const sessionGridSize = $derived<number | null>($sessionQuery.data?.session?.gameMap?.gridSize ?? null);

	type Position = { x: number; y: number };
	type Direction = 'UP' | 'DOWN' | 'LEFT' | 'RIGHT';
	type MoveHistoryEntry = {
		fromPosition: Position;
		toPosition: Position;
		success: boolean;
	};
	type Cell = { type: string; visited: boolean; id: string; allowedDirections: string[] };
	type GameState = {
		battery: number;
		maxBattery: number;
		score: number;
		victory: boolean;
		gameOver: boolean;
		totalMoves: number;
		resetCount: number;
		mapName: string;
		totalParks: number;
		message: string;
		fogEnabled: boolean;
		fogRadius: number;
		playerPos: Position;
		nearbyGrid: Cell[][];
		currentMoves: MoveHistoryEntry[];
	};

	function withStableFogState(nextState: GameState, previousState: GameState | null): GameState {
		if (!previousState) return nextState;
		if (!previousState.fogEnabled) return nextState;
		return {
			...nextState,
			fogEnabled: true,
			fogRadius: nextState.fogRadius > 0 ? nextState.fogRadius : previousState.fogRadius
		};
	}

	let liveState = $state<GameState | null>(null);
	let initialLoading = $state(true);
	let initialError = $state<string | null>(null);
	let animatedPos = $state<Position | null>(null);
	let animatedTrailKeys = $state<Set<string>>(new Set());
	let isAnimatingMoves = $state(false);
	let lastAnimationSignature = '';
	let lastAnimationMoveCount = 0;
	let isMoving = $state(false);
	let isResetting = $state(false);
	let isManualAnimating = $state(false);
	let moveError = $state<string | null>(null);
	let manualAnimationTimer: ReturnType<typeof setTimeout> | null = null;

	const gameState = $derived<GameState | null>(liveState);
	let fullGrid = $state<Cell[][] | null>(null);
	// Set by the home page when it auto-generated the fog password for this session.
	const generatedFogPassword = typeof localStorage !== 'undefined' ? (localStorage.getItem(`fogPassword:${sessionId}`) ?? '') : '';
	let gridPasswordInput = $state(generatedFogPassword);
	let appliedGridPassword = $state('');
	let fullGridError = $state<string | null>(null);
	let loadingFullGrid = $state(false);
	let autoLoadedNonFogGrid = $state(false);
	let fullGridMode = $state<'none' | 'auto' | 'password'>('none');
	const isUsingFullGrid = $derived(!!fullGrid);
	const displayPlayerPos = $derived(
		gameState?.fogEnabled && !isUsingFullGrid
			? (gameState.playerPos ?? null)
			: (animatedPos ?? gameState?.playerPos ?? null)
	);
	const activeGrid = $derived<Cell[][]>(fullGrid ?? gameState?.nearbyGrid ?? []);
	const showFogMaskBoard = $derived(!!gameState?.fogEnabled && !isUsingFullGrid && !!sessionGridSize);
	const boardSize = $derived<number>(showFogMaskBoard ? (sessionGridSize ?? 0) : activeGrid.length);
	const boardIndices = $derived<number[]>(Array.from({ length: boardSize }, (_, i) => i));
	const isLargeMap = $derived((boardSize ?? 0) >= 30);

	const isOver = $derived(!!gameState && (gameState.gameOver || gameState.victory));
	const outOfBattery = $derived(!!gameState && gameState.gameOver && !gameState.victory && gameState.battery <= 0);
	const batteryPct = $derived(
		gameState && gameState.maxBattery > 0 ? Math.max(0, Math.min(100, (gameState.battery / gameState.maxBattery) * 100)) : 0
	);
	const lowBattery = $derived(!!gameState && !isOver && batteryPct <= 25);
	// The server writes the crash site into the message, e.g. "... [Hit: water at (9,6)]".
	const crashCell = $derived.by<Position | null>(() => {
		if (!gameState?.gameOver || gameState.victory) return null;
		const m = /\((\d+),\s*(\d+)\)/.exec(gameState.message ?? '');
		return m ? { x: Number(m[1]), y: Number(m[2]) } : null;
	});
	const endDetail = $derived.by(() => {
		const msg = gameState?.message ?? '';
		return /\[(.+?)\]/.exec(msg)?.[1] ?? (outOfBattery ? 'The battery ran out away from a charger.' : msg);
	});

	let blockedCell = $state<Position | null>(null);
	let blockedTimer: ReturnType<typeof setTimeout> | null = null;
	let moveErrorTimer: ReturnType<typeof setTimeout> | null = null;
	let resetArmed = $state(false);
	let resetArmTimer: ReturnType<typeof setTimeout> | null = null;

	function flashBlocked(pos: Position) {
		if (blockedTimer) clearTimeout(blockedTimer);
		blockedCell = pos;
		blockedTimer = setTimeout(() => (blockedCell = null), 700);
	}

	function showMoveError(message: string) {
		if (moveErrorTimer) clearTimeout(moveErrorTimer);
		moveError = message;
		moveErrorTimer = setTimeout(() => (moveError = null), 3500);
	}

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
					liveState = withStableFogState(gs, liveState);
				},
				error(err) { console.error('WS error', err); },
				complete() {}
			}
		);

		return () => unsubscribe();
	});

	onMount(() => {
		let cancelled = false;

		const loadInitialState = async () => {
			initialLoading = true;
			initialError = null;
			try {
				const result = await client
					.query(gql(GAME_STATE_QUERY), { sessionID: sessionId }, { requestPolicy: 'network-only' })
					.toPromise();
				if (cancelled) return;
				if (result.error) throw result.error;
				const initialState = result.data?.gameState as GameState | undefined;
				if (!initialState) throw new Error('Session state is unavailable');
				liveState = withStableFogState(initialState, liveState);
			} catch (err) {
				if (cancelled) return;
				initialError = err instanceof Error ? err.message : 'Failed to load session state';
			} finally {
				if (!cancelled) initialLoading = false;
			}
		};

		void loadInitialState();
		return () => {
			cancelled = true;
		};
	});

	let promptCopied = $state(false);

	const llmPrompt = $derived(`Use this GraphQL API to control an existing Tesla Road Trip game session.

Goal: visit every park — query gameState.totalParks for the target count. Each successful move costs 1 battery; home (H) and superchargers (S) refill it. Moving into a building, water, or off the map ends the game immediately (the crash costs no battery). Moving the wrong way on a one-way road is simply rejected instead — no crash, no battery cost, keep driving. Do not rely on bumping into things to scout — read nearbyGrid's tile types before moving. Reaching 0 battery away from a charger also ends the game. Full rules and every field: ${typeof window !== 'undefined' ? window.location.origin : ''}/llms.txt

Session ID: ${sessionId}
GraphQL endpoint: ${typeof window !== 'undefined' ? window.location.origin : ''}/graphql
Playground: ${typeof window !== 'undefined' ? window.location.origin : ''}/playground
MCP endpoint: ${typeof window !== 'undefined' ? window.location.origin : ''}/mcp (Streamable HTTP transport)

To use MCP in Claude Code, run:
claude mcp add --transport http tesla-game ${typeof window !== 'undefined' ? window.location.origin : ''}/mcp

GraphQL introspection is enabled. Use the Playground Docs panel or query __schema/__type to discover fields before constructing operations.

## Inspect the API
query {
  __type(name: "GameState") {
    fields { name type { kind name ofType { kind name } } }
  }
}

## Read current session state (fog-safe)
query {
  gameState(sessionID: "${sessionId}") {
    mapName
    fogEnabled
    fogRadius
    playerPos { x y }
    battery
    maxBattery
    score
    totalParks
    batteryRisk
    victory
    gameOver
    message
    nearbyGrid { x y type visited id allowedDirections }
    visitedParks { id visited }
  }
}

## Read full grid (optional — only if you already know the grid password)
# Most sessions are meant to be played blind: do not try to guess the password.
# If you were not given one, skip this section and explore via nearbyGrid +
# move/bulkMove below instead. If fogEnabled=false, password is optional/ignored.
query {
  gameState(sessionID: "${sessionId}") {
    grid(password: "YOUR_GRID_PASSWORD") { type visited id allowedDirections }
  }
}

## Send one move
mutation {
  move(sessionID: "${sessionId}", direction: RIGHT) {
    success
    message
    attemptedTo { x y tileChar tileType passable }
    gameState {
      playerPos { x y }
      battery
      batteryRisk
      score
      victory
      gameOver
      fogRadius
      nearbyGrid { x y type visited id allowedDirections }
    }
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
    possibleMoves
    batteryRisk
    gameState {
      playerPos { x y }
      battery
      batteryRisk
      score
      victory
      gameOver
      fogRadius
      nearbyGrid { x y type visited id allowedDirections }
    }
  }
}

bulkMove accepts at most 50 moves per call. Check success, stoppedReason, stopReasonCode, truncated, gameOver, and victory before sending another operation; decide from success, gameOver, victory and the returned gameState, since the codes only explain why a run stopped.
batteryRisk is a hint based on the Manhattan distance to the nearest charger on the whole map, ignoring walls and one-way roads; plan battery against an actual path.
stopReasonCode "already_over" (or a move message starting "Game is already over") means the game had already ended; nothing moved. It is not a blocked path: call reset.

## Manage this session
mutation { reset(sessionID: "${sessionId}") { playerPos { x y } battery score victory gameOver fogRadius nearbyGrid { x y type visited id allowedDirections } } }
query { history(sessionID: "${sessionId}", page: 1, limit: 20, order: DESC) { totalMoves moves { moveNumber action success battery } } }
mutation { deleteSession(id: "${sessionId}") { message } }

Directions: UP DOWN LEFT RIGHT. RIGHT = x+1, DOWN = y+1. Full grid coordinates are grid[y][x].
One-way roads: a move must be listed (north/south/east/west) in allowedDirections of both the cell you leave and the cell you enter, when those lists are non-empty.
${gameState?.fogEnabled
	? `This session uses FOG (radius ${gameState.fogRadius}). nearbyGrid is the (2r+1)x(2r+1) window around the car; every cell carries its map coordinates in x and y, so key what you learn by those (off-map cells read as building, with off-map coordinates). Never select grid without the correct password: the error nulls the whole gameState response. If you already know the grid password, grid(password: ...) reveals the full map; otherwise skip it and explore via nearbyGrid/move/bulkMove — do not try to guess the password.`
	: `Fog is off: grid needs no password (the password argument is ignored). nearbyGrid is the 3x3 window around the car; every cell carries its map coordinates in x and y.`}`);

	function copyPrompt() {
		navigator.clipboard.writeText(llmPrompt);
		promptCopied = true;
		setTimeout(() => promptCopied = false, 2000);
	}

	async function fetchFullGrid(password: string): Promise<Cell[][]> {
		const result = await client
			.query(gql(FULL_GRID_QUERY), { sessionID: sessionId, password }, { requestPolicy: 'network-only' })
			.toPromise();
		if (result.error) throw result.error;
		const grid = result.data?.gameState?.grid as Cell[][] | undefined;
		if (!grid || !grid.length) throw new Error('No grid returned');
		return grid;
	}

	async function unlockFullGrid() {
		if (!gridPasswordInput.trim() || loadingFullGrid) return;
		loadingFullGrid = true;
		fullGridError = null;
		try {
			fullGrid = await fetchFullGrid(gridPasswordInput);
			appliedGridPassword = gridPasswordInput;
			fullGridMode = 'password';
		} catch (err) {
			fullGrid = null;
			fullGridMode = 'none';
			fullGridError = err instanceof Error ? err.message : 'Failed to unlock full grid';
		} finally {
			loadingFullGrid = false;
		}
	}

	function clearFullGrid() {
		fullGrid = null;
		appliedGridPassword = '';
		fullGridMode = 'none';
		fullGridError = null;
	}

	async function refreshFullGridIfUnlocked() {
		if (fullGridMode !== 'password') return;
		if (!appliedGridPassword) return;
		try {
			fullGrid = await fetchFullGrid(appliedGridPassword);
		} catch {
			fullGrid = null;
			fullGridMode = 'none';
		}
	}

	$effect(() => {
		const gs = gameState;
		if (!gs) return;
		if (gs.fogEnabled === true) {
			autoLoadedNonFogGrid = false;
			if (fullGridMode === 'auto') {
				fullGrid = null;
				fullGridMode = 'none';
			}
			return;
		}
		if (gs.fogEnabled !== false) return;
		if (fullGrid || loadingFullGrid || autoLoadedNonFogGrid) return;
		autoLoadedNonFogGrid = true;
		void (async () => {
			try {
				fullGrid = await fetchFullGrid('');
				fullGridMode = 'auto';
			} catch {
				// If this fails we fall back to nearbyGrid rendering.
			}
		})();
	});

	function keyToDirection(key: string): Direction | null {
		switch (key.toLowerCase()) {
			case 'arrowup':
				return 'UP';
			case 'arrowdown':
				return 'DOWN';
			case 'arrowleft':
				return 'LEFT';
			case 'arrowright':
				return 'RIGHT';
			default:
				return null;
		}
	}

	function isEditableTarget(target: EventTarget | null): boolean {
		if (!(target instanceof HTMLElement)) return false;
		const tagName = target.tagName.toLowerCase();
		return tagName === 'input' || tagName === 'textarea' || tagName === 'select' || target.isContentEditable;
	}

	function positionsEqual(a: Position | null | undefined, b: Position | null | undefined): boolean {
		return !!a && !!b && a.x === b.x && a.y === b.y;
	}

	function animateManualMove(from: Position, to: Position, nextState: GameState) {
		if (manualAnimationTimer) clearTimeout(manualAnimationTimer);

		isManualAnimating = true;
		isAnimatingMoves = true;
		animatedPos = from;
		animatedTrailKeys = new Set([`${from.x},${from.y}`]);

		manualAnimationTimer = setTimeout(() => {
			animatedPos = to;
			animatedTrailKeys = new Set([`${from.x},${from.y}`, `${to.x},${to.y}`]);

			manualAnimationTimer = setTimeout(() => {
				liveState = nextState;
				lastAnimationSignature = (nextState.currentMoves ?? [])
					.filter((move) => move.success)
					.map((move) => `${move.fromPosition.x},${move.fromPosition.y}>${move.toPosition.x},${move.toPosition.y}`)
					.join('|');
				lastAnimationMoveCount = (nextState.currentMoves ?? []).filter((move) => move.success).length;
				animatedPos = null;
				isAnimatingMoves = false;
				isManualAnimating = false;
				manualAnimationTimer = null;
			}, 180);
		}, 60);
	}

	async function sendMove(direction: Direction) {
		if (isMoving || isResetting || isManualAnimating || gameState?.gameOver || gameState?.victory) return;

		const from = displayPlayerPos ?? gameState?.playerPos ?? null;
		isMoving = true;
		moveError = null;

		try {
			const result = await client.mutation(gql(MOVE_MUTATION), { sessionID: sessionId, direction }).toPromise();
			if (result.error) throw result.error;

			const move = result.data?.move;
			if (!move) throw new Error('Move did not return a response');
			const nextState = move.gameState as GameState | null;
			const normalizedNextState = nextState ? withStableFogState(nextState, liveState) : null;
			const shouldAnimateManual = !!nextState && !(gameState?.fogEnabled && !isUsingFullGrid);
			if (move.success && normalizedNextState && from && !positionsEqual(from, normalizedNextState.playerPos) && shouldAnimateManual) {
				animateManualMove(from, normalizedNextState.playerPos, normalizedNextState);
			} else {
				if (normalizedNextState) liveState = normalizedNextState;
			}
			if (gameState?.fogEnabled === false) {
				await refreshFullGridIfUnlocked();
			}
			if (!move.success) {
				showMoveError(move.message || `Could not move ${direction.toLowerCase()}`);
				const at = move.attemptedTo;
				if (at && !normalizedNextState?.gameOver) flashBlocked({ x: at.x, y: at.y });
			}
		} catch (err) {
			showMoveError(err instanceof Error ? err.message : 'Move failed');
		} finally {
			isMoving = false;
		}
	}

	async function resetSession() {
		if (isMoving || isResetting) return;
		// Mid-run resets lose progress: require a second press. A finished game resets immediately.
		if (!isOver && !resetArmed) {
			resetArmed = true;
			if (resetArmTimer) clearTimeout(resetArmTimer);
			resetArmTimer = setTimeout(() => (resetArmed = false), 3000);
			return;
		}
		resetArmed = false;
		if (resetArmTimer) clearTimeout(resetArmTimer);

		isResetting = true;
		moveError = null;

		try {
			const result = await client.mutation(gql(RESET_MUTATION), { sessionID: sessionId }).toPromise();
			if (result.error) throw result.error;

			const resetState = result.data?.reset;
			if (!resetState) throw new Error('Reset did not return a game state');
			liveState = withStableFogState(resetState, liveState);
			blockedCell = null;
			animatedPos = null;
			animatedTrailKeys = new Set();
			isManualAnimating = false;
			if (manualAnimationTimer) clearTimeout(manualAnimationTimer);
			manualAnimationTimer = null;
			lastAnimationSignature = '';
			lastAnimationMoveCount = 0;
			if (gameState?.fogEnabled === false) {
				await refreshFullGridIfUnlocked();
			}
		} catch (err) {
			showMoveError(err instanceof Error ? err.message : 'Reset failed');
		} finally {
			isResetting = false;
		}
	}

	onMount(() => {
		function handleKeydown(event: KeyboardEvent) {
			if (isEditableTarget(event.target)) return;

			if (event.key.toLowerCase() === 'r') {
				event.preventDefault();
				void resetSession();
				return;
			}

			const direction = keyToDirection(event.key);
			if (!direction) return;
			if (gameState?.gameOver || gameState?.victory) return;

			event.preventDefault();
			void sendMove(direction);
		}

		window.addEventListener('keydown', handleKeydown);
		return () => {
			window.removeEventListener('keydown', handleKeydown);
			if (manualAnimationTimer) clearTimeout(manualAnimationTimer);
		};
	});

	function cellTextClass(cell: { type: string; allowedDirections?: string[] }): string {
		return cell.type === 'road' && hasDirections(cell) ? 'text-orange-500 font-bold' : '';
	}

	function cellColorClass(cell: Cell): string {
		switch (cell.type) {
			case 'home': return 'bg-red-500 border-red-200';
			case 'park': return cell.visited ? 'bg-emerald-800 border-emerald-700' : 'bg-emerald-500 border-emerald-200';
			case 'supercharger': return 'bg-yellow-400 border-yellow-200';
			case 'water': return 'bg-blue-400 border-blue-200';
			case 'building': return 'bg-slate-700 border-slate-600';
			default: return 'bg-white border-gray-100';
		}
	}

	// Glyphs keep tile types readable without relying on colour alone.
	function tileGlyph(cell: Cell): string {
		switch (cell.type) {
			case 'home': return '⌂';
			case 'park': return cell.visited ? '✓' : '🌳';
			case 'supercharger': return '⚡';
			case 'water': return '≈';
			default: return '';
		}
	}

	function tileGlyphClass(cell: Cell): string {
		switch (cell.type) {
			case 'home':
			case 'water': return 'text-white font-bold';
			case 'park': return cell.visited ? 'text-white font-bold' : '';
			default: return '';
		}
	}

	function cellLabel(cell: Cell | null, isPlayer: boolean): string {
		if (!cell) return isPlayer ? 'car' : 'hidden by fog';
		return `${isPlayer ? 'car on ' : ''}${cell.type}${cell.visited ? ', visited' : ''}`;
	}

	$effect(() => {
		if (isManualAnimating) return;
		if (showFogMaskBoard) {
			animatedPos = null;
			animatedTrailKeys = new Set();
			isAnimatingMoves = false;
			lastAnimationSignature = '';
			lastAnimationMoveCount = 0;
			return;
		}
		const moves = gameState?.currentMoves?.filter((move) => move.success) ?? [];
		if (!gameState || moves.length === 0) {
			animatedPos = null;
			animatedTrailKeys = new Set();
			isAnimatingMoves = false;
			lastAnimationSignature = '';
			lastAnimationMoveCount = 0;
			return;
		}

		const signature = moves
			.map((move) => `${move.fromPosition.x},${move.fromPosition.y}>${move.toPosition.x},${move.toPosition.y}`)
			.join('|');
		if (signature === lastAnimationSignature) return;
		const previousSignature = lastAnimationSignature;
		const previousMoveCount = lastAnimationMoveCount;
		lastAnimationSignature = signature;
		lastAnimationMoveCount = moves.length;

		const isAppendOnly = previousSignature !== '' && signature.startsWith(`${previousSignature}|`);
		if (!isAppendOnly) {
			animatedPos = null;
			animatedTrailKeys = new Set();
			isAnimatingMoves = false;
			return;
		}

		const deltaMoves = moves.slice(previousMoveCount);
		if (deltaMoves.length === 0) return;
		const historicalMoves = moves.slice(0, previousMoveCount);

		let cancelled = false;
		let index = 0;
		let trail = new Set<string>();
		for (const move of historicalMoves) {
			trail.add(`${move.fromPosition.x},${move.fromPosition.y}`);
			trail.add(`${move.toPosition.x},${move.toPosition.y}`);
		}
		isAnimatingMoves = true;
		animatedPos = deltaMoves[0].fromPosition;
		trail.add(`${deltaMoves[0].fromPosition.x},${deltaMoves[0].fromPosition.y}`);
		animatedTrailKeys = new Set(trail);

		const step = () => {
			if (cancelled) return;
			const move = deltaMoves[index];
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

	function toNearbyIndex(x: number, y: number): { ix: number; iy: number } | null {
		if (!gameState) return null;
		// nearbyGrid is centered on the authoritative server position, not the animated client position.
		const center = gameState.playerPos;
		const radius = gameState.fogRadius > 0 ? gameState.fogRadius : 1;
		const minX = center.x - radius;
		const minY = center.y - radius;
		const ix = x - minX;
		const iy = y - minY;
		if (iy < 0 || iy >= gameState.nearbyGrid.length) return null;
		if (ix < 0 || ix >= (gameState.nearbyGrid[iy]?.length ?? 0)) return null;
		return { ix, iy };
	}

	function getRenderedCell(x: number, y: number): Cell | null {
		if (isUsingFullGrid) {
			return fullGrid?.[y]?.[x] ?? null;
		}
		if (showFogMaskBoard) {
			const idx = toNearbyIndex(x, y);
			if (!idx || !gameState) return null;
			return gameState.nearbyGrid[idx.iy]?.[idx.ix] ?? null;
		}
		return activeGrid[y]?.[x] ?? null;
	}

	function isPlayerCell(x: number, y: number): boolean {
		if (!displayPlayerPos) return false;
		if (isUsingFullGrid) {
			return x === displayPlayerPos.x && y === displayPlayerPos.y;
		}
		if (showFogMaskBoard && gameState?.playerPos) {
			return x === gameState.playerPos.x && y === gameState.playerPos.y;
		}
		const centerY = Math.floor((activeGrid.length - 1) / 2);
		const centerX = activeGrid[centerY] ? Math.floor((activeGrid[centerY].length - 1) / 2) : 0;
		return x === centerX && y === centerY;
	}

	function isTrailCell(x: number, y: number): boolean {
		if (!(isUsingFullGrid || showFogMaskBoard)) return false;
		return trailKeys.has(`${x},${y}`);
	}
</script>

<svelte:head>
	<title>{sessionDisplayName ?? sessionId} — Tesla Road Trip</title>
</svelte:head>

<div class="max-w-[1900px] mx-auto px-3 sm:px-4 py-4 lg:py-6">
	<div class={`grid grid-cols-1 gap-4 xl:gap-6 items-start ${isLargeMap ? '' : 'lg:grid-cols-[minmax(0,1fr)_22rem] xl:grid-cols-[minmax(0,1fr)_26rem]'}`}>
		<!-- left: header, status, board -->
		<section class="min-w-0 bg-white rounded-2xl border border-[#e8e8e8] shadow-sm">
			<div class="p-3 sm:p-4 border-b border-gray-100">
				<div class="flex flex-wrap items-start justify-between gap-3">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2 text-xs uppercase tracking-widest text-gray-500">
							<span>Session</span>
							{#if gameState}
								<span>·</span>
								<span class="normal-case tracking-normal font-mono">{gameState.mapName}</span>
								{#if gameState.fogEnabled}
									<span class="normal-case tracking-normal inline-flex items-center rounded-full bg-blue-100 text-blue-800 px-2 py-0.5 text-xs">🌫 Fog r{gameState.fogRadius}</span>
								{/if}
							{/if}
						</div>
						<div class="mt-1 flex items-center gap-2">
							{#if sessionDisplayName}
								<span class="text-lg leading-none text-gray-800 font-medium">{sessionDisplayName}</span>
								<button
									onclick={() => navigator.clipboard.writeText(sessionId)}
									class="font-mono text-sm leading-none text-gray-500 hover:text-blue-700 transition-colors"
									title="Copy session ID"
								>({sessionId})</button>
							{:else}
								<button
									onclick={() => navigator.clipboard.writeText(sessionId)}
									class="font-mono text-lg leading-none text-gray-800 hover:text-blue-700 transition-colors"
									title="Copy session ID"
								>{sessionId}</button>
							{/if}
						</div>
					</div>

					{#if gameState}
						<div class="grid grid-cols-3 gap-2 text-center">
							<div class="bg-gray-50 rounded-xl px-4 py-2 min-w-[5.5rem]">
								<div class="text-xl font-medium leading-tight text-gray-800">{gameState.score}<span class="text-gray-500 font-light">/{gameState.totalParks}</span></div>
								<div class="text-xs text-gray-600">Parks</div>
							</div>
							<div class="bg-gray-50 rounded-xl px-4 py-2 min-w-[5.5rem]">
								<div class="text-xl font-light leading-tight text-gray-800">{gameState.totalMoves}</div>
								<div class="text-xs text-gray-600">Moves</div>
							</div>
							<div class="bg-gray-50 rounded-xl px-4 py-2 min-w-[5.5rem]">
								<div class="text-xl font-light leading-tight text-gray-800">{gameState.resetCount}</div>
								<div class="text-xs text-gray-600">Resets</div>
							</div>
						</div>
					{:else}
						<p class="text-sm text-gray-500">Loading…</p>
					{/if}
				</div>

				{#if gameState}
					<div class="mt-4">
						<div class="flex items-center justify-between text-sm mb-1.5">
							<span class="font-medium text-gray-700">⚡ Battery</span>
							<span class="tabular-nums font-medium {lowBattery || outOfBattery ? 'text-red-700' : 'text-gray-700'}">{gameState.battery} / {gameState.maxBattery}</span>
						</div>
						<div
							class="h-3.5 bg-gray-100 rounded-full overflow-hidden"
							role="progressbar"
							aria-label="Battery"
							aria-valuemin={0}
							aria-valuemax={gameState.maxBattery}
							aria-valuenow={gameState.battery}
						>
							<div
								class="h-full rounded-full transition-all duration-300 {batteryPct > 50 ? 'bg-green-500' : batteryPct > 25 ? 'bg-orange-500' : 'bg-red-500'} {lowBattery ? 'low-battery' : ''}"
								style="width: {batteryPct}%"
							></div>
						</div>
					</div>
				{/if}
			</div>

			{#if gameState}
				{#if isOver}
				<div class="px-3 sm:px-4 pt-3 sm:pt-4">
					{#if gameState.victory}
						<div role="status" class="status-bar flex flex-wrap items-center justify-between gap-3 rounded-xl border border-green-300 bg-green-50 px-4 py-3">
							<div>
								<p class="font-medium text-green-900">🏆 You won!</p>
								<p class="text-sm text-green-900">All {gameState.totalParks} parks in {gameState.totalMoves} moves{gameState.resetCount > 0 ? ` · ${gameState.resetCount} reset${gameState.resetCount === 1 ? '' : 's'}` : ''}.</p>
							</div>
							<button type="button" onclick={resetSession} disabled={isResetting}
								class="text-sm px-4 py-2 rounded-full bg-green-700 text-white hover:bg-green-800 transition-colors disabled:opacity-50">
								{isResetting ? 'Resetting…' : 'Play again'} <kbd class="kbd-inline" aria-hidden="true">R</kbd>
							</button>
						</div>
					{:else if gameState.gameOver}
						<div role="status" class="status-bar flex flex-wrap items-center justify-between gap-3 rounded-xl border border-red-300 bg-red-50 px-4 py-3">
							<div>
								<p class="font-medium text-red-900">{outOfBattery ? '🪫 Out of battery' : '💥 Crashed'}</p>
								<p class="text-sm text-red-900">{endDetail}</p>
							</div>
							<button type="button" onclick={resetSession} disabled={isResetting}
								class="text-sm px-4 py-2 rounded-full bg-red-700 text-white hover:bg-red-800 transition-colors disabled:opacity-50">
								{isResetting ? 'Resetting…' : 'Try again'} <kbd class="kbd-inline" aria-hidden="true">R</kbd>
							</button>
						</div>
					{/if}
				</div>
				{/if}
				<div class="sr-only" aria-live="polite">
					Battery {gameState.battery} of {gameState.maxBattery}. Parks {gameState.score} of {gameState.totalParks}.
					{#if gameState.victory}You won.{:else if gameState.gameOver}Game over. {endDetail}{/if}
					{moveError ?? ''}
				</div>
			{/if}

			<div class="p-3 sm:p-4 flex items-start justify-center board-pane">
				{#if boardSize > 0}
					<table class="game-board border-collapse" role="grid" aria-label="Game board" style={`--grid-size: ${boardSize}; --board-width: ${isLargeMap ? '92vw' : 'calc(100vw - 30rem)'}`}>
						<tbody>
						{#each boardIndices as y}
							<tr>
								{#each boardIndices as x}
									{@const cell = getRenderedCell(x, y)}
									{@const isPlayer = isPlayerCell(x, y)}
									{@const isTrail = isTrailCell(x, y)}
									{@const isCrash = crashCell?.x === x && crashCell?.y === y}
									{@const isBlocked = blockedCell?.x === x && blockedCell?.y === y}
									<td class="game-cell text-center border transition-colors
										{cell ? cellColorClass(cell) : 'bg-slate-300 border-slate-300'}
										{isTrail && !isPlayer ? 'ring-2 ring-inset ring-sky-300' : ''}
										{cell?.visited && !isPlayer && cell.type !== 'park' ? 'opacity-60' : ''}"
										class:crash-cell={isCrash}
										class:blocked-flash={isBlocked}
										role="gridcell"
										aria-label={cellLabel(cell, isPlayer)}>
										{#if isPlayer}
											{gameState?.gameOver && !gameState.victory && !crashCell ? '💥' : '🚗'}
										{:else if isCrash}
											💥
										{:else if isTrail}
											<span class="text-sky-500 leading-none">•</span>
										{:else if cell && tileGlyph(cell)}
											<span class="leading-none {tileGlyphClass(cell)}">{tileGlyph(cell)}</span>
										{:else if cell && hasDirections(cell)}
											<span class={`leading-none ${cellTextClass(cell)}`}>{directionGlyph(cell.allowedDirections)}</span>
										{/if}
									</td>
								{/each}
							</tr>
						{/each}
						</tbody>
					</table>
				{:else if initialLoading}
					<div class="flex items-center justify-center h-64 text-gray-500">
						<div class="text-center">
							<span class="text-4xl block mb-3">🚗</span>
							<p class="text-sm">Loading <code class="font-mono">{sessionId}</code>…</p>
						</div>
					</div>
				{:else if initialError}
					<div class="flex items-center justify-center h-64 text-red-700">
						<p class="text-sm">Session not found: {initialError}</p>
					</div>
				{:else}
					<div class="flex items-center justify-center h-64 text-gray-500">
						<div class="text-center">
							<span class="text-4xl block mb-3">🚗</span>
							<p class="text-sm">Waiting for moves on <code class="font-mono">{sessionId}</code>…</p>
							<p class="text-xs mt-2">Point an AI at this session to see it play</p>
						</div>
					</div>
				{/if}
			</div>

			<div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs text-gray-600 px-4 pb-4">
				<div class="flex flex-wrap items-center gap-x-4 gap-y-1">
					<span class="flex items-center gap-1"><span class="inline-flex items-center justify-center w-4 h-4 rounded-sm bg-red-500 text-white font-bold leading-none">⌂</span> Home</span>
					<span class="flex items-center gap-1"><span class="inline-flex items-center justify-center w-4 h-4 rounded-sm bg-emerald-500 text-[10px] leading-none">🌳</span> Park</span>
					<span class="flex items-center gap-1"><span class="inline-flex items-center justify-center w-4 h-4 rounded-sm bg-yellow-400 text-[10px] leading-none">⚡</span> Charger</span>
					<span class="flex items-center gap-1"><span class="inline-flex items-center justify-center w-4 h-4 rounded-sm bg-blue-400 text-white font-bold leading-none">≈</span> Water</span>
					<span class="flex items-center gap-1"><span class="inline-block w-4 h-4 rounded-sm bg-slate-700"></span> Blocked</span>
					<span class="flex items-center gap-1"><span class="text-sky-500">•</span> Trail</span>
				</div>
				<a href="/lobby" class="hover:text-gray-900 underline-offset-2 hover:underline transition-colors">← Back to sessions</a>
			</div>
		</section>

		<!-- right: controls + AI -->
		<aside class={`min-w-0 space-y-3 ${isLargeMap ? '' : 'lg:sticky lg:top-4'}`}>
			{#if gameState}
				<div class="rounded-2xl bg-white border border-[#e8e8e8] shadow-sm p-4">
					<h2 class="text-xs font-semibold uppercase tracking-widest text-gray-600">Controls</h2>
					<div class="mt-3 flex items-center gap-5">
						<div class="grid grid-cols-3 gap-1 shrink-0" role="group" aria-label="Drive controls">
							{#each [['', ''], ['UP', '↑'], ['', ''], ['LEFT', '←'], ['DOWN', '↓'], ['RIGHT', '→']] as [dir, glyph]}
								{#if dir}
									<button
										type="button"
										aria-label={`Drive ${dir.toLowerCase()}`}
										onclick={() => sendMove(dir as Direction)}
										disabled={isOver || isMoving || isResetting || isManualAnimating}
										class="dpad-btn"
									>{glyph}</button>
								{:else}
									<span></span>
								{/if}
							{/each}
						</div>
						<div class="min-w-0 text-xs text-gray-600 space-y-2">
							<p><kbd class="kbd">↑</kbd><kbd class="kbd">↓</kbd><kbd class="kbd">←</kbd><kbd class="kbd">→</kbd> to drive</p>
							<p>Keys are ignored while typing in a field.</p>
							{#if moveError}
								<p class="text-amber-800">⚠ {moveError}</p>
							{/if}
						</div>
					</div>
					<div class="mt-4">
						<button
							type="button"
							onclick={resetSession}
							disabled={isMoving || isResetting}
							class="text-sm px-4 py-1.5 rounded-full border transition-colors disabled:opacity-40 disabled:cursor-not-allowed
								{resetArmed ? 'bg-red-700 border-red-700 text-white hover:bg-red-800' : 'border-red-300 text-red-700 hover:bg-red-50'}"
						>
							{isResetting ? 'Resetting…' : resetArmed ? 'Click again to confirm' : 'Reset session'}
							{#if !resetArmed && !isResetting}<kbd class="kbd-inline-light" aria-hidden="true">R</kbd>{/if}
						</button>
					</div>
				</div>

				{#if gameState?.fogEnabled}
					<div class="rounded-2xl bg-white border border-[#e8e8e8] shadow-sm p-4">
						<h2 class="text-xs font-semibold uppercase tracking-widest text-gray-600">Fog mode</h2>
						<p class="text-xs text-gray-600 mt-1">This session hides the map beyond {gameState.fogRadius} cell{gameState.fogRadius === 1 ? '' : 's'} of the car. Enter the password chosen at creation to reveal the full map.</p>
						{#if generatedFogPassword}
							<p class="text-xs text-gray-700 mt-2">
								Auto-generated password:
								<button type="button" onclick={() => navigator.clipboard.writeText(generatedFogPassword)} title="Copy password" class="font-mono font-semibold underline decoration-dotted hover:text-blue-700">{generatedFogPassword}</button>
							</p>
						{/if}
						<div class="mt-3 flex flex-wrap items-center gap-2">
							<input
								type="text"
								bind:value={gridPasswordInput}
								placeholder="Grid password"
								aria-label="Grid password"
								class="min-w-[10rem] flex-1 border border-gray-300 rounded-lg px-3 py-1.5 text-xs bg-white focus-visible:outline-2 focus-visible:outline-blue-600"
							/>
							<button
								type="button"
								onclick={unlockFullGrid}
								disabled={loadingFullGrid || !gridPasswordInput.trim()}
								class="text-xs px-3 py-1.5 rounded-full border border-blue-300 text-blue-700 hover:bg-blue-50 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
							>
								{loadingFullGrid ? 'Unlocking…' : 'Unlock full grid'}
							</button>
							{#if isUsingFullGrid}
								<button
									type="button"
									onclick={clearFullGrid}
									class="text-xs px-3 py-1.5 rounded-full border border-gray-300 text-gray-700 hover:bg-gray-50 transition-colors"
								>
									Use nearby grid
								</button>
							{/if}
						</div>
						{#if appliedGridPassword && isUsingFullGrid}
							<p class="text-xs text-green-800 mt-2">Full grid unlocked for this session.</p>
						{/if}
						{#if fullGridError}
							<p class="text-xs text-red-700 mt-2">{fullGridError}</p>
						{/if}
					</div>
				{/if}
			{/if}

			<div class="rounded-2xl bg-white border border-[#e8e8e8] shadow-sm p-4">
				<div class="flex items-start justify-between gap-3">
					<div class="min-w-0">
						<h2 class="text-xs font-semibold uppercase tracking-widest text-gray-600">Play with an AI</h2>
						<p class="mt-1 text-sm text-gray-700">Copy the prompt into an AI chat to let it drive session <code class="font-mono font-semibold">{sessionId}</code>.</p>
					</div>
					<button
						onclick={copyPrompt}
						class="text-sm px-4 py-2 rounded-full border transition-colors shrink-0 {promptCopied ? 'bg-green-50 border-green-300 text-green-800' : 'border-blue-300 text-blue-700 hover:bg-blue-50'}"
					>{promptCopied ? 'Copied!' : 'Copy'}</button>
				</div>
				<details class="mt-3 group">
					<summary class="cursor-pointer text-xs font-medium text-gray-600 hover:text-gray-900 select-none">Show full prompt</summary>
					<textarea
						readonly
						aria-label="Prompt for an AI"
						value={llmPrompt}
						class="mt-2 w-full text-xs font-mono text-gray-700 bg-gray-50 rounded-xl p-3 resize-none prompt-pane focus-visible:outline-2 focus-visible:outline-blue-600 leading-relaxed border border-gray-100"
						onclick={(e) => (e.target as HTMLTextAreaElement).select()}
					></textarea>
				</details>
			</div>
		</aside>
	</div>
</div>

<style>
	.board-pane {
		min-height: calc(100vh - 24rem);
		overflow: visible;
	}

	.prompt-pane {
		height: 24rem;
	}

	.status-bar {
		min-height: 4rem;
	}

	.game-board {
		--cell-size: clamp(
			1.75rem,
			min(
				calc((var(--board-width) - 5rem) / var(--grid-size)),
				calc((100vh - 23rem) / var(--grid-size))
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

	.crash-cell {
		position: relative;
		z-index: 1;
		outline: 3px solid #b91c1c;
		outline-offset: -3px;
		animation: crash-pulse 1s ease-in-out infinite;
	}

	.blocked-flash {
		position: relative;
		z-index: 1;
		outline: 3px solid #f59e0b;
		outline-offset: -3px;
		animation: shake 0.35s ease-in-out;
	}

	.low-battery {
		animation: crash-pulse 1.2s ease-in-out infinite;
	}

	@keyframes crash-pulse {
		50% { opacity: 0.55; }
	}

	@keyframes shake {
		0%, 100% { transform: translateX(0); }
		25% { transform: translateX(-3px); }
		75% { transform: translateX(3px); }
	}

	@media (prefers-reduced-motion: reduce) {
		.crash-cell,
		.blocked-flash,
		.low-battery {
			animation: none;
		}
	}

	.kbd {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.35rem;
		height: 1.35rem;
		margin: 0 0.08rem;
		border-radius: 0.35rem;
		border: 1px solid #d1d5db;
		background: #f9fafb;
		color: #374151;
		font-size: 0.7rem;
		font-weight: 600;
		line-height: 1;
	}

	.kbd-inline,
	.kbd-inline-light {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 1.2rem;
		height: 1.2rem;
		margin-left: 0.35rem;
		border-radius: 0.3rem;
		font-size: 0.65rem;
		font-weight: 600;
		line-height: 1;
	}

	.kbd-inline {
		background: rgba(255, 255, 255, 0.25);
	}

	.kbd-inline-light {
		border: 1px solid #fca5a5;
	}

	.dpad-btn {
		width: 2.5rem;
		height: 2.5rem;
		border-radius: 0.6rem;
		border: 1px solid #d1d5db;
		background: #f9fafb;
		color: #374151;
		font-size: 1.1rem;
		line-height: 1;
		transition: background-color 0.15s;
	}

	.dpad-btn:hover:not(:disabled) {
		background: #e5e7eb;
	}

	.dpad-btn:focus-visible {
		outline: 2px solid #2563eb;
		outline-offset: 2px;
	}

	.dpad-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* Stacked layout below lg: the board gets the full width. */
	@media (max-width: 1023px) {
		.game-board {
			--cell-size: clamp(1.75rem, calc((100vw - 4rem) / var(--grid-size)), 3.25rem);
		}
	}
</style>
