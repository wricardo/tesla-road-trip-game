import type { CellConfigEntry } from '$lib/directional';

export type MapSummary = {
	mapId: string;
	name: string;
	description: string;
	gridSize: number;
	maxBattery: number;
};

export type LegendEntry = { key: string; value: string };

export type MapLayout = {
	name: string;
	description: string;
	gridSize: number;
	maxBattery: number;
	startingBattery: number;
	layout: string[];
	legend: LegendEntry[];
	cellConfigs: CellConfigEntry[];
};

/** Human label for a map: raw slugs ("bayou_braids") become "Bayou Braids". */
export function prettyMapName(name: string): string {
	const n = (name ?? '').trim();
	if (!n) return '';
	return n.includes('_') || n === n.toLowerCase()
		? n.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
		: n;
}

/** Display name for a map id, using the maps list when available. */
export function mapLabel(mapId: string, maps: Pick<MapSummary, 'mapId' | 'name'>[] = []): string {
	const m = maps.find((x) => x.mapId === mapId);
	return prettyMapName(m?.name || mapId);
}

/**
 * The one map order used everywhere (home, maps page, next map): maps carry no
 * difficulty field, so grid size stands in for it (small → large), then name.
 */
export function sortMaps<T extends Pick<MapSummary, 'mapId' | 'name' | 'gridSize'>>(maps: T[]): T[] {
	return [...maps].sort(
		(a, b) =>
			a.gridSize - b.gridSize ||
			prettyMapName(a.name).localeCompare(prettyMapName(b.name)) ||
			a.mapId.localeCompare(b.mapId)
	);
}

/** Next map after `mapId` in sortMaps order (wraps to the first), or null. */
export function nextMap<T extends Pick<MapSummary, 'mapId' | 'name' | 'gridSize'>>(maps: T[], mapId: string): T | null {
	const sorted = sortMaps(maps);
	if (sorted.length === 0) return null;
	const i = sorted.findIndex((m) => m.mapId === mapId);
	const next = sorted[(i + 1) % sorted.length];
	return next && next.mapId !== mapId ? next : null;
}

export function tileType(char: string, legend: LegendEntry[] = [], cellConfigs: CellConfigEntry[] = []): string {
	return cellConfigs.find((entry) => entry.key === char)?.type ?? legend.find((entry) => entry.key === char)?.value ?? 'road';
}

export function tileClass(type: string): string {
	switch (type) {
		case 'home': return 'bg-red-500';
		case 'park': return 'bg-emerald-500';
		case 'supercharger': return 'bg-yellow-400';
		case 'water': return 'bg-blue-400';
		case 'building': return 'bg-slate-700';
		default: return 'bg-white';
	}
}

export function countTiles(map: MapLayout | undefined, type: string): number {
	if (!map) return 0;
	let n = 0;
	for (const row of map.layout) for (const ch of row) if (tileType(ch, map.legend, map.cellConfigs) === type) n++;
	return n;
}
