export type Direction = 'UP' | 'DOWN' | 'LEFT' | 'RIGHT';
export type CrashCell = { type: string } | null | undefined;
export type CrashKind = 'water' | 'building' | 'edge';

const DELTAS: Record<Direction, [number, number]> = {
	UP: [0, -1],
	DOWN: [0, 1],
	LEFT: [-1, 0],
	RIGHT: [1, 0]
};

export const DIRECTIONS: Direction[] = ['UP', 'DOWN', 'LEFT', 'RIGHT'];

/**
 * Which directions would end the run, based only on what the client already knows.
 * `cellAt(x, y)` returns the known cell, or undefined when the tile is hidden (fog):
 * unknown tiles are never flagged. `gridSize` (when known) detects the map edge.
 */
export function crashDirections(
	pos: { x: number; y: number } | null | undefined,
	cellAt: (x: number, y: number) => CrashCell,
	gridSize: number | null | undefined
): Partial<Record<Direction, CrashKind>> {
	const out: Partial<Record<Direction, CrashKind>> = {};
	if (!pos) return out;
	for (const dir of DIRECTIONS) {
		const [dx, dy] = DELTAS[dir];
		const x = pos.x + dx;
		const y = pos.y + dy;
		if (gridSize && (x < 0 || y < 0 || x >= gridSize || y >= gridSize)) {
			out[dir] = 'edge';
			continue;
		}
		const cell = cellAt(x, y);
		if (cell?.type === 'water' || cell?.type === 'building') out[dir] = cell.type;
	}
	return out;
}

export function crashWarning(kind: CrashKind): string {
	return kind === 'edge' ? 'would crash off the edge of the map' : `would crash into ${kind}`;
}
