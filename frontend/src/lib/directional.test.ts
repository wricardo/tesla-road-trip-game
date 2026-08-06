import { describe, expect, it } from 'vitest';
import { directionGlyph, directionsForChar, directionalTitle, hasDirections } from './directional';

describe('directional utilities', () => {
	it('maps known directional chars to allowed directions', () => {
		expect(directionsForChar('|')).toEqual(['north', 'south']);
		expect(directionsForChar('>')).toEqual(['east']);
		expect(directionsForChar('x')).toEqual([]);
	});

	it('returns expected glyphs and fallbacks', () => {
		expect(directionGlyph(['north', 'south'])).toBe('↕');
		expect(directionGlyph(['east', 'west'])).toBe('↔');
		expect(directionGlyph(['south', 'west'])).toBe('↙');
		expect(directionGlyph(['north', 'south', 'east'])).toBe('•');
		expect(directionGlyph([])).toBe('');
	});

	it('formats directional titles', () => {
		expect(directionalTitle(['north', 'east'])).toBe('road: north, east only');
		expect(directionalTitle([])).toBe('road');
	});

	it('detects whether a cell has direction constraints', () => {
		expect(hasDirections({ allowedDirections: ['north'] })).toBe(true);
		expect(hasDirections({ allowedDirections: [] })).toBe(false);
		expect(hasDirections({})).toBe(false);
		expect(hasDirections(null)).toBe(false);
		expect(hasDirections(undefined)).toBe(false);
	});
});
