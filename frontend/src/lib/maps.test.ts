import { describe, expect, it } from 'vitest';
import { mapLabel, nextMap, prettyMapName, sortMaps } from './maps';

const maps = [
	{ mapId: 'hub_maze', name: 'hub_maze', gridSize: 20 },
	{ mapId: 'easy_trapped_park', name: 'Easy Mode - Trapped Park', gridSize: 10 },
	{ mapId: 'easy', name: 'Easy Mode', gridSize: 10 },
	{ mapId: 'classic', name: 'Classic Layout', gridSize: 15 }
];

describe('prettyMapName', () => {
	it('title-cases raw slugs and keeps real names', () => {
		expect(prettyMapName('bayou_braids')).toBe('Bayou Braids');
		expect(prettyMapName('strategic')).toBe('Strategic');
		expect(prettyMapName("Sarah's Circuit")).toBe("Sarah's Circuit");
	});
});

describe('sortMaps', () => {
	it('orders by grid size, then name', () => {
		expect(sortMaps(maps).map((m) => m.mapId)).toEqual(['easy', 'easy_trapped_park', 'classic', 'hub_maze']);
	});
	it('does not mutate its input', () => {
		const copy = [...maps];
		sortMaps(maps);
		expect(maps).toEqual(copy);
	});
});

describe('nextMap', () => {
	it('returns the following map and wraps around', () => {
		expect(nextMap(maps, 'easy')?.mapId).toBe('easy_trapped_park');
		expect(nextMap(maps, 'hub_maze')?.mapId).toBe('easy');
	});
	it('returns null when there is nothing else to play', () => {
		expect(nextMap([maps[0]], 'hub_maze')).toBeNull();
		expect(nextMap([], 'easy')).toBeNull();
	});
});

describe('mapLabel', () => {
	it('uses the list name, falling back to the prettified id', () => {
		expect(mapLabel('classic', maps)).toBe('Classic Layout');
		expect(mapLabel('hub_maze', maps)).toBe('Hub Maze');
		expect(mapLabel('keys_run', maps)).toBe('Keys Run');
	});
});
