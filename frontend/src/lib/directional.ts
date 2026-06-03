export type DirectionalCell = {
	type?: string;
	allowedDirections?: string[];
};

export type CellConfigEntry = {
	key: string;
	type: string;
	allowedDirections: string[];
};

export const standardDirectionalCellConfigs: CellConfigEntry[] = [
	{ key: '|', type: 'road', allowedDirections: ['north', 'south'] },
	{ key: '-', type: 'road', allowedDirections: ['east', 'west'] },
	{ key: '^', type: 'road', allowedDirections: ['north'] },
	{ key: 'v', type: 'road', allowedDirections: ['south'] },
	{ key: '>', type: 'road', allowedDirections: ['east'] },
	{ key: '<', type: 'road', allowedDirections: ['west'] },
	{ key: 'J', type: 'road', allowedDirections: ['north', 'east'] },
	{ key: 'L', type: 'road', allowedDirections: ['north', 'west'] },
	{ key: '7', type: 'road', allowedDirections: ['south', 'east'] },
	{ key: 'r', type: 'road', allowedDirections: ['south', 'west'] }
];

export const directionalChars = new Set(standardDirectionalCellConfigs.map((entry) => entry.key));

export function directionsForChar(char: string, cellConfigs: CellConfigEntry[] = standardDirectionalCellConfigs): string[] {
	return cellConfigs.find((entry) => entry.key === char)?.allowedDirections ?? [];
}

export function directionGlyph(dirs: string[] = []): string {
	const set = new Set(dirs);
	const n = set.has('north');
	const s = set.has('south');
	const e = set.has('east');
	const w = set.has('west');
	if (n && s && !e && !w) return '↕';
	if (e && w && !n && !s) return '↔';
	if (n && !s && !e && !w) return '↑';
	if (s && !n && !e && !w) return '↓';
	if (e && !n && !s && !w) return '→';
	if (w && !n && !s && !e) return '←';
	if (n && e && !s && !w) return '↗';
	if (n && w && !s && !e) return '↖';
	if (s && e && !n && !w) return '↘';
	if (s && w && !n && !e) return '↙';
	return dirs.length > 0 ? '•' : '';
}

export function directionalTitle(dirs: string[] = []): string {
	return dirs.length > 0 ? `road: ${dirs.join(', ')} only` : 'road';
}

export function hasDirections(cell: DirectionalCell | null | undefined): boolean {
	return (cell?.allowedDirections?.length ?? 0) > 0;
}
