import { describe, expect, it } from 'vitest';
import { absoluteTime, relativeTime } from './time';

const now = new Date('2026-10-06T12:00:00Z').getTime();

describe('relativeTime', () => {
	it('formats recent times relative to now', () => {
		expect(relativeTime(new Date(now - 20_000).toISOString(), now)).toBe('just now');
		expect(relativeTime(new Date(now - 2 * 60_000).toISOString(), now)).toBe('2m ago');
		expect(relativeTime(new Date(now - 3 * 3_600_000).toISOString(), now)).toBe('3h ago');
		expect(relativeTime(new Date(now - 2 * 86_400_000).toISOString(), now)).toBe('2d ago');
	});
	it('falls back to a short date after a week', () => {
		expect(relativeTime('2026-09-20T12:00:00Z', now)).toBe('Sep 20');
	});
	it('handles missing or invalid input', () => {
		expect(relativeTime('', now)).toBe('—');
		expect(relativeTime('nope', now)).toBe('—');
		expect(relativeTime(null, now)).toBe('—');
	});
});

describe('absoluteTime', () => {
	it('returns empty for invalid input', () => {
		expect(absoluteTime('nope')).toBe('');
		expect(absoluteTime('2026-09-20T12:00:00Z')).not.toBe('');
	});
});
