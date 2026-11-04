import { describe, expect, it } from 'vitest';
import { DEFAULT_MOVE_DELAY_MS } from '$lib/createSession';
import { DEFAULT_START_SETTINGS, validateStartSettings } from './startSettings';

describe('start settings defaults', () => {
	it('starts with fog enabled at radius 1', () => {
		expect(DEFAULT_START_SETTINGS.fogEnabled).toBe(true);
		expect(DEFAULT_START_SETTINGS.fogRadius).toBe(1);
		expect(DEFAULT_START_SETTINGS.gridPassword).toBe('');
		expect(DEFAULT_START_SETTINGS.moveDelayMs).toBe(DEFAULT_MOVE_DELAY_MS);
	});

	it('accepts the defaults', () => {
		expect(validateStartSettings(DEFAULT_START_SETTINGS)).toBeNull();
	});
});

describe('validateStartSettings', () => {
	it('requires a whole fog radius of at least 1', () => {
		expect(validateStartSettings({ ...DEFAULT_START_SETTINGS, fogRadius: 0 })).toContain('at least 1');
		expect(validateStartSettings({ ...DEFAULT_START_SETTINGS, fogRadius: -3 })).toContain('at least 1');
		expect(validateStartSettings({ ...DEFAULT_START_SETTINGS, fogRadius: 1.5 })).toContain('whole number');
	});

	it('rejects a negative or non-numeric move delay but allows 0', () => {
		expect(validateStartSettings({ ...DEFAULT_START_SETTINGS, moveDelayMs: -1 })).toContain('0 or greater');
		expect(validateStartSettings({ ...DEFAULT_START_SETTINGS, moveDelayMs: Number.NaN })).toContain('0 or greater');
		expect(validateStartSettings({ ...DEFAULT_START_SETTINGS, moveDelayMs: 0 })).toBeNull();
	});
});
