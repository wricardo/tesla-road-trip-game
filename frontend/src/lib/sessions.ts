export type SessionStatus = 'playing' | 'won' | 'lost';
export type StatusFilter = 'all' | SessionStatus;
export type SessionSort = 'recent' | 'moves' | 'parks';

export type SessionSummary = {
	id: string;
	displayName: string | null;
	mapName: string;
	lastActionAt: string;
	score: number;
	totalMoves: number;
	victory: boolean;
	gameOver: boolean;
};

export function sessionStatus(s: { victory: boolean; gameOver: boolean }): SessionStatus {
	return s.victory ? 'won' : s.gameOver ? 'lost' : 'playing';
}

export function statusCounts(sessions: SessionSummary[]): Record<StatusFilter, number> {
	const c = { all: sessions.length, playing: 0, won: 0, lost: 0 };
	for (const s of sessions) c[sessionStatus(s)]++;
	return c;
}

function ts(s: SessionSummary): number {
	const t = new Date(s.lastActionAt).getTime();
	return Number.isNaN(t) ? 0 : t;
}

/** Filter by status chip and free-text (name, id or map), then sort. */
export function filterSessions<T extends SessionSummary>(
	sessions: T[],
	opts: { status?: StatusFilter; search?: string; sort?: SessionSort; mapLabel?: (mapId: string) => string } = {}
): T[] {
	const { status = 'all', search = '', sort = 'recent', mapLabel = (m) => m } = opts;
	const q = search.trim().toLowerCase();
	return sessions
		.filter((s) => status === 'all' || sessionStatus(s) === status)
		.filter((s) => !q || `${s.displayName ?? ''} ${s.id} ${s.mapName} ${mapLabel(s.mapName)}`.toLowerCase().includes(q))
		.sort((a, b) =>
			sort === 'moves' ? b.totalMoves - a.totalMoves || ts(b) - ts(a)
			: sort === 'parks' ? b.score - a.score || ts(b) - ts(a)
			: ts(b) - ts(a)
		);
}

/** Home page "Recent sessions": started ones only, newest first, at most `limit`. */
export function recentSessions<T extends SessionSummary>(sessions: T[], limit = 8): T[] {
	return filterSessions(sessions.filter((s) => s.totalMoves > 0), { sort: 'recent' }).slice(0, limit);
}
