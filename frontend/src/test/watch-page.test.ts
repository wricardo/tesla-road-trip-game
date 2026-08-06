import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import WatchPage from '../routes/watch/[id]/+page.svelte';

type Position = { x: number; y: number };
type Cell = { type: string; visited: boolean; id: string; allowedDirections: string[] };
type MoveEntry = { fromPosition: Position; toPosition: Position; success: boolean };
type GameState = {
	battery: number;
	maxBattery: number;
	score: number;
	victory: boolean;
	gameOver: boolean;
	totalMoves: number;
	mapName: string;
	fogEnabled: boolean;
	fogRadius: number;
	playerPos: Position;
	nearbyGrid: Cell[][];
	currentMoves: MoveEntry[];
};

const mockRuntime = vi.hoisted(() => ({
	wsSinks: [] as Array<{ next: (data: unknown) => void }>,
	queryImpl: ((_query: unknown, _variables: Record<string, unknown>) => ({ data: {} })) as (
		query: unknown,
		variables: Record<string, unknown>
	) => Promise<{ data?: unknown; error?: Error | null }> | { data?: unknown; error?: Error | null },
	mutationImpl: ((_query: unknown, _variables: Record<string, unknown>) => ({ data: {} })) as (
		query: unknown,
		variables: Record<string, unknown>
	) => Promise<{ data?: unknown; error?: Error | null }> | { data?: unknown; error?: Error | null },
	sessionGridSize: 3,
	sessionDisplayName: 'Session 6d2e',
	sessionMapName: 'classic',
	queryCalls: [] as Array<{ query: unknown; variables: Record<string, unknown> }>,
	mutationCalls: [] as Array<{ query: unknown; variables: Record<string, unknown> }>
}));

vi.mock('$app/stores', async () => {
	const { readable } = await import('svelte/store');
	return {
		page: readable({
			params: { id: '6d2e' },
			url: new URL('http://localhost:8000/watch/6d2e')
		})
	};
});

vi.mock('@urql/svelte', () => {
	const fakeClient = {
		query: (query: unknown, variables: Record<string, unknown>) => {
			mockRuntime.queryCalls.push({ query, variables });
			return {
				toPromise: async () => await mockRuntime.queryImpl(query, variables)
			};
		},
		mutation: (query: unknown, variables: Record<string, unknown>) => {
			mockRuntime.mutationCalls.push({ query, variables });
			return {
				toPromise: async () => await mockRuntime.mutationImpl(query, variables)
			};
		}
	};

	return {
		gql: (query: string) => query,
		getContextClient: () => fakeClient,
		queryStore: () => ({
			subscribe: (run: (value: unknown) => void) => {
				run({
					data: {
						session: {
							id: '6d2e',
							displayName: mockRuntime.sessionDisplayName,
							mapName: mockRuntime.sessionMapName,
							gameMap: { gridSize: mockRuntime.sessionGridSize }
						}
					}
				});
				return () => {};
			}
		})
	};
});

vi.mock('graphql-ws', () => ({
	createClient: () => ({
		subscribe: (_payload: unknown, sink: { next: (data: unknown) => void }) => {
			mockRuntime.wsSinks.push(sink);
			return () => {};
		}
	})
}));

function makeNearbyGrid(): Cell[][] {
	return [
		[
			{ type: 'road', visited: false, id: '0,0', allowedDirections: [] },
			{ type: 'road', visited: false, id: '1,0', allowedDirections: [] },
			{ type: 'road', visited: false, id: '2,0', allowedDirections: [] }
		],
		[
			{ type: 'road', visited: false, id: '0,1', allowedDirections: [] },
			{ type: 'home', visited: true, id: '1,1', allowedDirections: [] },
			{ type: 'park', visited: false, id: '2,1', allowedDirections: [] }
		],
		[
			{ type: 'road', visited: false, id: '0,2', allowedDirections: [] },
			{ type: 'road', visited: false, id: '1,2', allowedDirections: [] },
			{ type: 'supercharger', visited: false, id: '2,2', allowedDirections: [] }
		]
	];
}

function makeFullGrid5x5(): Cell[][] {
	return Array.from({ length: 5 }, (_, y) =>
		Array.from({ length: 5 }, (_, x) => ({
			type: x === 2 && y === 2 ? 'home' : 'road',
			visited: false,
			id: `${x},${y}`,
			allowedDirections: []
		}))
	);
}

function makeGameState(overrides: Partial<GameState> = {}): GameState {
	return {
		battery: 8,
		maxBattery: 10,
		score: 1,
		victory: false,
		gameOver: false,
		totalMoves: 2,
		mapName: 'classic',
		fogEnabled: true,
		fogRadius: 1,
		playerPos: { x: 1, y: 1 },
		nearbyGrid: makeNearbyGrid(),
		currentMoves: [],
		...overrides
	};
}

function deferred<T>() {
	let resolve!: (value: T) => void;
	let reject!: (reason?: unknown) => void;
	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

function findCarCellIndex(container: HTMLElement): number {
	return Array.from(container.querySelectorAll('td')).findIndex((cell) => cell.textContent?.includes('🚗'));
}

describe('watch page', () => {
	beforeEach(() => {
		mockRuntime.wsSinks = [];
		mockRuntime.queryCalls = [];
		mockRuntime.mutationCalls = [];
		mockRuntime.sessionGridSize = 3;
		mockRuntime.sessionDisplayName = 'Session 6d2e';
		mockRuntime.sessionMapName = 'classic';
		vi.restoreAllMocks();
		vi.stubGlobal('confirm', vi.fn(() => true));
		Object.defineProperty(globalThis.navigator, 'clipboard', {
			value: {
				writeText: vi.fn().mockResolvedValue(undefined)
			},
			configurable: true
		});
		mockRuntime.queryImpl = (query, variables) => {
			if (typeof query === 'string' && query.includes('query GameState')) {
				return { data: { gameState: makeGameState() }, error: null };
			}
			if (typeof query === 'string' && query.includes('query FullGrid')) {
				return { data: { gameState: { grid: makeNearbyGrid() } }, error: null };
			}
			throw new Error(`Unexpected query: ${String(query)} with ${JSON.stringify(variables)}`);
		};
		mockRuntime.mutationImpl = () => ({ data: {}, error: null });
	});

	afterEach(() => {
		cleanup();
		vi.useRealTimers();
	});

	it('loads initial state and applies websocket updates', async () => {
		render(WatchPage);

		await screen.findByText('8/10');
		expect(mockRuntime.wsSinks).toHaveLength(1);

		mockRuntime.wsSinks[0].next({
			data: {
				sessionUpdated: makeGameState({ battery: 7, score: 2, totalMoves: 3 })
			}
		});

		await screen.findByText('7/10');
	});

	it('sends keyboard moves and ignores keydown from editable elements', async () => {
		mockRuntime.mutationImpl = (query, variables) => {
			if (typeof query === 'string' && query.includes('mutation Move')) {
				const direction = String(variables.direction ?? 'RIGHT');
				return {
					data: {
						move: {
							success: true,
							message: '',
							gameState: makeGameState({
								totalMoves: 3,
								playerPos: direction === 'RIGHT' ? { x: 2, y: 1 } : { x: 1, y: 1 }
							})
						}
					},
					error: null
				};
			}
			throw new Error(`Unexpected mutation: ${String(query)}`);
		};

		render(WatchPage);
		await screen.findByText('Reset session');

		await fireEvent.keyDown(window, { key: 'ArrowRight' });

		await waitFor(() => {
			expect(mockRuntime.mutationCalls.some((call) => typeof call.query === 'string' && call.query.includes('mutation Move'))).toBe(true);
		});

		const moveCallCount = mockRuntime.mutationCalls.length;
		const passwordInput = screen.getByPlaceholderText('Grid password');
		await fireEvent.keyDown(passwordInput, { key: 'ArrowUp' });

		expect(mockRuntime.mutationCalls).toHaveLength(moveCallCount);
	});

	it('resets session from keyboard shortcut when confirmed', async () => {
		mockRuntime.mutationImpl = (query) => {
			if (typeof query === 'string' && query.includes('mutation Reset')) {
				return {
					data: {
						reset: makeGameState({ battery: 10, score: 0, totalMoves: 0, playerPos: { x: 0, y: 0 } })
					},
					error: null
				};
			}
			if (typeof query === 'string' && query.includes('mutation Move')) {
				return { data: { move: { success: true, message: '', gameState: makeGameState() } }, error: null };
			}
			throw new Error(`Unexpected mutation: ${String(query)}`);
		};

		render(WatchPage);
		await screen.findByText('Reset session');

		await fireEvent.keyDown(window, { key: 'r' });

		await waitFor(() => {
			expect(globalThis.confirm as unknown as ReturnType<typeof vi.fn>).toHaveBeenCalledTimes(1);
			expect(mockRuntime.mutationCalls.some((call) => typeof call.query === 'string' && call.query.includes('mutation Reset'))).toBe(true);
		});
	});

	it('unlocks and clears full grid view with password flow', async () => {
		mockRuntime.queryImpl = (query, variables) => {
			if (typeof query === 'string' && query.includes('query GameState')) {
				return { data: { gameState: makeGameState() }, error: null };
			}
			if (typeof query === 'string' && query.includes('query FullGrid')) {
				if (variables.password === 'secret') {
					return { data: { gameState: { grid: makeNearbyGrid() } }, error: null };
				}
				return { error: new Error('bad password') };
			}
			throw new Error(`Unexpected query: ${String(query)}`);
		};

		render(WatchPage);
		await screen.findByText('Unlock full grid');

		const passwordInput = screen.getByPlaceholderText('Grid password');
		await fireEvent.input(passwordInput, { target: { value: 'secret' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Unlock full grid' }));

		await screen.findByText('Full grid unlocked for this session.');
		expect(
			mockRuntime.queryCalls.some(
				(call) =>
					typeof call.query === 'string' &&
					call.query.includes('query FullGrid') &&
					call.variables.password === 'secret'
			)
		).toBe(true);

		await fireEvent.click(screen.getByRole('button', { name: 'Use nearby grid' }));
		await waitFor(() => {
			expect(screen.queryByText('Full grid unlocked for this session.')).not.toBeInTheDocument();
		});
	});

	it('copies the LLM prompt and resets copied state', async () => {
		render(WatchPage);
		await screen.findByText('Prompt for LLM');
		vi.useFakeTimers();

		const copyButton = screen.getByRole('button', { name: 'Copy' });
		await fireEvent.click(copyButton);

		expect(navigator.clipboard.writeText).toHaveBeenCalledTimes(1);
		expect(screen.getByRole('button', { name: 'Copied!' })).toBeInTheDocument();

		await vi.advanceTimersByTimeAsync(2000);

		expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument();
	});

	it('renders active, won, and crashed status from game state updates', async () => {
		render(WatchPage);
		await screen.findByText('Active');
		expect(screen.getByText('🟢')).toBeInTheDocument();

		mockRuntime.wsSinks[0].next({
			data: {
				sessionUpdated: makeGameState({ victory: true, gameOver: false })
			}
		});
		await screen.findByText('Won');
		expect(screen.getByText('🏆')).toBeInTheDocument();

		mockRuntime.wsSinks[0].next({
			data: {
				sessionUpdated: makeGameState({ victory: false, gameOver: true })
			}
		});
		await screen.findByText('Crashed');
		expect(screen.getAllByText('💥').length).toBeGreaterThan(0);
	});

	it('renders reset button disabled during move and reset requests', async () => {
		const movePending = deferred<{ data: unknown; error: null }>();
		const resetPending = deferred<{ data: unknown; error: null }>();

		mockRuntime.mutationImpl = (query) => {
			if (typeof query === 'string' && query.includes('mutation Move')) {
				return movePending.promise;
			}
			if (typeof query === 'string' && query.includes('mutation Reset')) {
				return resetPending.promise;
			}
			throw new Error(`Unexpected mutation: ${String(query)}`);
		};

		render(WatchPage);
		const resetButton = await screen.findByRole('button', { name: 'Reset session' });

		await fireEvent.keyDown(window, { key: 'ArrowRight' });
		await waitFor(() => {
			expect(resetButton).toBeDisabled();
			expect(screen.getByText('moving…')).toBeInTheDocument();
		});

		movePending.resolve({
			data: { move: { success: true, message: '', gameState: makeGameState({ playerPos: { x: 2, y: 1 } }) } },
			error: null
		});
		await waitFor(() => {
			expect(resetButton).not.toBeDisabled();
			expect(screen.getByText('ready')).toBeInTheDocument();
		});

		await fireEvent.click(resetButton);
		await waitFor(() => {
			expect(resetButton).toBeDisabled();
			expect(screen.getByText('resetting…')).toBeInTheDocument();
			expect(screen.getByRole('button', { name: 'Resetting…' })).toBeDisabled();
		});

		resetPending.resolve({
			data: { reset: makeGameState({ battery: 10, score: 0, totalMoves: 0 }) },
			error: null
		});
		await waitFor(() => {
			expect(screen.getByRole('button', { name: 'Reset session' })).not.toBeDisabled();
			expect(screen.getByText('ready')).toBeInTheDocument();
		});
	});

	it('renders fog-mode board using session grid size and avoids auto full-grid fetch', async () => {
		mockRuntime.sessionGridSize = 5;
		mockRuntime.queryImpl = (query, variables) => {
			if (typeof query === 'string' && query.includes('query GameState')) {
				return { data: { gameState: makeGameState({ fogEnabled: true, fogRadius: 2 }) }, error: null };
			}
			if (typeof query === 'string' && query.includes('query FullGrid')) {
				if (variables.password === 'secret') {
					return { data: { gameState: { grid: makeNearbyGrid() } }, error: null };
				}
				return { error: new Error('bad password') };
			}
			throw new Error(`Unexpected query: ${String(query)} with ${JSON.stringify(variables)}`);
		};

		const { container } = render(WatchPage);
		await screen.findByText('🌫 Fog r2');

		expect(container.querySelectorAll('td')).toHaveLength(25);
		expect(
			mockRuntime.queryCalls.some(
				(call) => typeof call.query === 'string' && call.query.includes('query FullGrid')
			)
		).toBe(false);

		const passwordInput = screen.getByPlaceholderText('Grid password');
		await fireEvent.input(passwordInput, { target: { value: 'secret' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Unlock full grid' }));
		await screen.findByText('Full grid unlocked for this session.');
	});

	it('renders player movement on fog board when moving by keyboard', async () => {
		mockRuntime.sessionGridSize = 5;
		mockRuntime.queryImpl = (query) => {
			if (typeof query === 'string' && query.includes('query GameState')) {
				return { data: { gameState: makeGameState({ fogEnabled: true, fogRadius: 1, playerPos: { x: 2, y: 2 } }) }, error: null };
			}
			if (typeof query === 'string' && query.includes('query FullGrid')) {
				return { data: { gameState: { grid: makeNearbyGrid() } }, error: null };
			}
			throw new Error(`Unexpected query: ${String(query)}`);
		};
		mockRuntime.mutationImpl = (query, variables) => {
			if (typeof query === 'string' && query.includes('mutation Move')) {
				return {
					data: {
						move: {
							success: true,
							message: '',
							gameState: makeGameState({
								// Simulate backend omitting/losing fog flag in move responses.
								fogEnabled: false,
								fogRadius: 1,
								playerPos: variables.direction === 'RIGHT' ? { x: 3, y: 2 } : { x: 2, y: 2 },
								currentMoves: [
									{
										fromPosition: { x: 2, y: 2 },
										toPosition: { x: 3, y: 2 },
										success: true
									}
								]
							})
						}
					},
					error: null
				};
			}
			throw new Error(`Unexpected mutation: ${String(query)}`);
		};

		const { container } = render(WatchPage);
		await screen.findByText('🌫 Fog r1');
		await screen.findByText('Reset session');

		expect(container.querySelectorAll('td')).toHaveLength(25);
		expect(findCarCellIndex(container)).toBe(12);

		await fireEvent.keyDown(window, { key: 'ArrowRight' });

		await waitFor(() => {
			expect(
				mockRuntime.mutationCalls.some(
					(call) =>
						typeof call.query === 'string' &&
						call.query.includes('mutation Move') &&
						call.variables.direction === 'RIGHT'
				)
			).toBe(true);
			expect(findCarCellIndex(container)).toBe(13);
			expect(screen.getByText('🌫 Fog r1')).toBeInTheDocument();
			expect(container.querySelectorAll('td')).toHaveLength(25);
		});
	});

	it('animates keyboard movement in fog sessions when full grid is unlocked', async () => {
		vi.useFakeTimers();
		mockRuntime.queryImpl = (query, variables) => {
			if (typeof query === 'string' && query.includes('query GameState')) {
				return { data: { gameState: makeGameState({ fogEnabled: true, fogRadius: 1, playerPos: { x: 1, y: 1 } }) }, error: null };
			}
			if (typeof query === 'string' && query.includes('query FullGrid')) {
				if (variables.password === 'secret') {
					return { data: { gameState: { grid: makeNearbyGrid() } }, error: null };
				}
				return { error: new Error('bad password') };
			}
			throw new Error(`Unexpected query: ${String(query)}`);
		};
		mockRuntime.mutationImpl = (query) => {
			if (typeof query === 'string' && query.includes('mutation Move')) {
				return {
					data: {
						move: {
							success: true,
							message: '',
							gameState: makeGameState({
								fogEnabled: true,
								fogRadius: 1,
								playerPos: { x: 2, y: 1 },
								currentMoves: [
									{
										fromPosition: { x: 1, y: 1 },
										toPosition: { x: 2, y: 1 },
										success: true
									}
								]
							})
						}
					},
					error: null
				};
			}
			throw new Error(`Unexpected mutation: ${String(query)}`);
		};

		const { container } = render(WatchPage);
		await screen.findByText('🌫 Fog r1');

		const passwordInput = screen.getByPlaceholderText('Grid password');
		await fireEvent.input(passwordInput, { target: { value: 'secret' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Unlock full grid' }));
		await screen.findByText('Full grid unlocked for this session.');
		expect(findCarCellIndex(container)).toBe(4);

		await fireEvent.keyDown(window, { key: 'ArrowRight' });
		await vi.advanceTimersByTimeAsync(70);

		expect(findCarCellIndex(container)).toBe(5);
	});

	it('does not replay trace animation from origin on websocket move updates', async () => {
		vi.useFakeTimers();
		mockRuntime.queryImpl = (query) => {
			if (typeof query === 'string' && query.includes('query GameState')) {
				return {
					data: {
						gameState: makeGameState({
							fogEnabled: false,
							playerPos: { x: 2, y: 2 },
							currentMoves: [
								{ fromPosition: { x: 0, y: 2 }, toPosition: { x: 1, y: 2 }, success: true },
								{ fromPosition: { x: 1, y: 2 }, toPosition: { x: 2, y: 2 }, success: true }
							]
						})
					},
					error: null
				};
			}
			if (typeof query === 'string' && query.includes('query FullGrid')) {
				return { data: { gameState: { grid: makeFullGrid5x5() } }, error: null };
			}
			throw new Error(`Unexpected query: ${String(query)}`);
		};

		const { container } = render(WatchPage);
		await screen.findByText('Use nearby grid');
		expect(findCarCellIndex(container)).toBe(12);
		expect(mockRuntime.wsSinks).toHaveLength(1);

		mockRuntime.wsSinks[0].next({
			data: {
				sessionUpdated: makeGameState({
					fogEnabled: false,
					playerPos: { x: 3, y: 2 },
					currentMoves: [
						{ fromPosition: { x: 0, y: 2 }, toPosition: { x: 1, y: 2 }, success: true },
						{ fromPosition: { x: 1, y: 2 }, toPosition: { x: 2, y: 2 }, success: true },
						{ fromPosition: { x: 2, y: 2 }, toPosition: { x: 3, y: 2 }, success: true }
					]
				})
			}
		});

		await waitFor(() => {
			// Should begin at the newest move origin (2,2), not replay from (0,2).
			expect(findCarCellIndex(container)).toBe(12);
		});
		expect(screen.getAllByText('•').length).toBeGreaterThan(0);

		await vi.advanceTimersByTimeAsync(220);
		expect(findCarCellIndex(container)).toBe(13);
	});
});
