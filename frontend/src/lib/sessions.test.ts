import { describe, expect, it } from 'vitest';
import { filterSessions, recentSessions, sessionStatus, statusCounts, type SessionSummary } from './sessions';

const s = (id: string, o: Partial<SessionSummary> = {}): SessionSummary => ({
	id,
	displayName: null,
	mapName: 'classic',
	lastActionAt: '2026-10-06T10:00:00Z',
	score: 0,
	totalMoves: 1,
	victory: false,
	gameOver: false,
	...o
});

const list = [
	s('a1', { displayName: 'Claude run', totalMoves: 5, lastActionAt: '2026-10-06T09:00:00Z' }),
	s('b2', { victory: true, score: 3, totalMoves: 30, lastActionAt: '2026-10-06T11:00:00Z' }),
	s('c3', { gameOver: true, mapName: 'hub_maze', totalMoves: 12, lastActionAt: '2026-10-06T10:00:00Z' }),
	s('d4', { totalMoves: 0, lastActionAt: '2026-10-06T11:30:00Z' })
];

describe('sessions helpers', () => {
	it('derives status', () => {
		expect(sessionStatus({ victory: true, gameOver: true })).toBe('won');
		expect(sessionStatus({ victory: false, gameOver: true })).toBe('lost');
		expect(sessionStatus({ victory: false, gameOver: false })).toBe('playing');
	});

	it('counts each chip', () => {
		expect(statusCounts(list)).toEqual({ all: 4, playing: 2, won: 1, lost: 1 });
	});

	it('filters by status and searches name, id and map label', () => {
		expect(filterSessions(list, { status: 'lost' }).map((x) => x.id)).toEqual(['c3']);
		expect(filterSessions(list, { search: 'claude' }).map((x) => x.id)).toEqual(['a1']);
		expect(filterSessions(list, { search: 'hub maze', mapLabel: () => 'Hub Maze' }).length).toBe(4);
		expect(filterSessions(list, { search: 'b2' }).map((x) => x.id)).toEqual(['b2']);
	});

	it('sorts by last activity or moves', () => {
		expect(filterSessions(list).map((x) => x.id)).toEqual(['d4', 'b2', 'c3', 'a1']);
		expect(filterSessions(list, { sort: 'moves' }).map((x) => x.id)).toEqual(['b2', 'c3', 'a1', 'd4']);
	});

	it('recentSessions hides never-started sessions and caps the list', () => {
		expect(recentSessions(list).map((x) => x.id)).toEqual(['b2', 'c3', 'a1']);
		expect(recentSessions(list, 1).map((x) => x.id)).toEqual(['b2']);
	});
});
