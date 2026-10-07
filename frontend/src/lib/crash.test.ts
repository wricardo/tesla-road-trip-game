import { describe, expect, it } from 'vitest';
import { crashDirections, crashWarning } from './crash';

// 3x3 map: water above the car, building to the right, road elsewhere.
const grid = [
	['road', 'water', 'road'],
	['road', 'road', 'building'],
	['road', 'road', 'road']
].map((row) => row.map((type) => ({ type })));
const cellAt = (x: number, y: number) => grid[y]?.[x];

describe('crashDirections', () => {
	it('flags water and buildings next to the car', () => {
		expect(crashDirections({ x: 1, y: 1 }, cellAt, 3)).toEqual({ UP: 'water', RIGHT: 'building' });
	});

	it('flags the map edge when the size is known', () => {
		expect(crashDirections({ x: 0, y: 2 }, cellAt, 3)).toEqual({ DOWN: 'edge', LEFT: 'edge' });
	});

	it('never flags tiles the client cannot see', () => {
		expect(crashDirections({ x: 1, y: 1 }, () => undefined, null)).toEqual({});
		expect(crashDirections(null, cellAt, 3)).toEqual({});
	});

	it('describes the warning', () => {
		expect(crashWarning('building')).toBe('would crash into building');
		expect(crashWarning('edge')).toBe('would crash off the edge of the map');
	});
});
