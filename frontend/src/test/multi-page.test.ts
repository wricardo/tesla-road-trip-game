import { cleanup, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MultiPage from '../routes/multi/+page.svelte';

const runtime = vi.hoisted(() => ({
	fetchCalls: [] as string[],
	sessionFogEnabled: true,
	sessionFogRadius: 2,
	mapGridSize: 5,
	wsSinks: [] as Array<{ next: (payload: unknown) => void }>,
	fetchResponse: () => ({
		data: {
			gameState: {
				battery: 17,
				maxBattery: 22,
				score: 1,
				victory: false,
				gameOver: false,
				totalMoves: 420,
				fogEnabled: true,
				fogRadius: 2,
				playerPos: { x: 2, y: 2 },
				nearbyGrid: nearby5x5(),
				currentMoves: []
			}
		}
	})
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$app/stores', async () => {
	const { readable } = await import('svelte/store');
	return {
		page: readable({
			url: new URL('http://localhost:8000/multi?map=medium_downtown&sessions=6d2e')
		})
	};
});

vi.mock('@urql/svelte', () => ({
	gql: (query: string) => query,
	getContextClient: () => ({}),
	queryStore: ({ query }: { query: string }) => ({
		subscribe: (run: (value: unknown) => void) => {
			if (query.includes('query Sessions')) {
				run({
					data: {
						sessions: {
							sessions: [
								{
									id: '6d2e',
									mapName: 'medium_downtown',
									gameState: { fogEnabled: runtime.sessionFogEnabled, fogRadius: runtime.sessionFogRadius }
								}
							]
						}
					},
					reexecute: vi.fn()
				});
			} else {
				run({
					data: {
						maps: [{ mapId: 'medium_downtown', name: 'Downtown Grid', gridSize: runtime.mapGridSize }]
					}
				});
			}
			return () => {};
		},
		reexecute: vi.fn()
	})
}));

vi.mock('graphql-ws', () => ({
	createClient: () => ({
		subscribe: (_payload: unknown, sink: { next: (payload: unknown) => void }) => {
			runtime.wsSinks.push(sink);
			return () => {};
		}
	})
}));

function nearby5x5() {
	return Array.from({ length: 5 }, (_, y) =>
		Array.from({ length: 5 }, (_, x) => ({
			type: x === 2 && y === 2 ? 'home' : 'road',
			visited: false,
			id: `${x},${y}`,
			allowedDirections: []
		}))
	);
}

function fullGrid5x5() {
	return Array.from({ length: 5 }, (_, y) =>
		Array.from({ length: 5 }, (_, x) => ({
			type: x === 2 && y === 2 ? 'home' : 'road',
			visited: false,
			id: `${x},${y}`,
			allowedDirections: []
		}))
	);
}

function findCarCellIndex(container: HTMLElement): number {
	return Array.from(container.querySelectorAll('td')).findIndex((cell) => cell.textContent?.includes('🚗'));
}

describe('multi page fog rendering', () => {
	beforeEach(() => {
		runtime.fetchCalls = [];
		runtime.sessionFogEnabled = true;
		runtime.sessionFogRadius = 2;
		runtime.mapGridSize = 5;
		runtime.wsSinks = [];
		runtime.fetchResponse = () => ({
			data: {
				gameState: {
					battery: 17,
					maxBattery: 22,
					score: 1,
					victory: false,
					gameOver: false,
					totalMoves: 420,
					fogEnabled: true,
					fogRadius: 2,
					playerPos: { x: 2, y: 2 },
					nearbyGrid: nearby5x5(),
					currentMoves: []
				}
			}
		});
		vi.stubGlobal(
			'fetch',
			vi.fn(async (_url: string, init?: RequestInit) => {
				runtime.fetchCalls.push(String(init?.body ?? ''));
				return {
					json: async () => runtime.fetchResponse()
				} as Response;
			})
		);
	});

	afterEach(() => {
		cleanup();
		vi.unstubAllGlobals();
	});

	it('renders fog sessions without full-grid query failures', async () => {
		const { container } = render(MultiPage);

		await screen.findByText('1 parks · 420 moves');
		expect(screen.queryByText('Loading…')).not.toBeInTheDocument();
		expect(container.querySelectorAll('td')).toHaveLength(25);

		await waitFor(() => {
			expect(runtime.fetchCalls.length).toBeGreaterThan(0);
		});
		expect(runtime.fetchCalls[0]).toContain('nearbyGrid');
		expect(runtime.fetchCalls[0]).not.toContain('grid { type visited id allowedDirections }');
	});

	it('animates only newly appended moves instead of replaying from start', async () => {
		vi.useFakeTimers();
		runtime.sessionFogEnabled = false;
		runtime.fetchResponse = () => ({
			data: {
				gameState: {
					battery: 10,
					maxBattery: 22,
					score: 1,
					victory: false,
					gameOver: false,
					totalMoves: 2,
					fogEnabled: false,
					fogRadius: 1,
					playerPos: { x: 2, y: 2 },
					grid: fullGrid5x5(),
					currentMoves: [
						{ fromPosition: { x: 0, y: 2 }, toPosition: { x: 1, y: 2 }, success: true },
						{ fromPosition: { x: 1, y: 2 }, toPosition: { x: 2, y: 2 }, success: true }
					]
				}
			}
		});

		const { container } = render(MultiPage);
		await screen.findByText('1 parks · 2 moves');
		expect(findCarCellIndex(container)).toBe(12);

		runtime.wsSinks[0].next({
			data: {
				sessionUpdated: {
					battery: 9,
					maxBattery: 22,
					score: 1,
					victory: false,
					gameOver: false,
					totalMoves: 3,
					fogEnabled: false,
					fogRadius: 1,
					playerPos: { x: 3, y: 2 },
					grid: fullGrid5x5(),
					currentMoves: [
						{ fromPosition: { x: 0, y: 2 }, toPosition: { x: 1, y: 2 }, success: true },
						{ fromPosition: { x: 1, y: 2 }, toPosition: { x: 2, y: 2 }, success: true },
						{ fromPosition: { x: 2, y: 2 }, toPosition: { x: 3, y: 2 }, success: true }
					]
				}
			}
		});

		await waitFor(() => {
			// Start animating from the newest move origin (x=2,y=2), not the path origin (x=0,y=2).
			expect(findCarCellIndex(container)).toBe(12);
		});
		expect(container.querySelectorAll('span.inline-block.rounded-full.w-1.h-1.shrink-0').length).toBeGreaterThan(0);

		await vi.advanceTimersByTimeAsync(220);
		expect(findCarCellIndex(container)).toBe(13);
	});
});
