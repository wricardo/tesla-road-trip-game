import { beforeEach, describe, expect, it } from 'vitest';
import {
	canEditMap,
	forgetMapEditKey,
	getMapEditKey,
	mapSaveErrorMessage,
	setMapEditKey
} from './mapEditKeys';

describe('map edit keys', () => {
	beforeEach(() => {
		window.localStorage.clear();
	});

	it('stores and reads a key per map', () => {
		setMapEditKey('mine', 'token-a');
		setMapEditKey('other', 'token-b');

		expect(getMapEditKey('mine')).toBe('token-a');
		expect(getMapEditKey('other')).toBe('token-b');
		expect(getMapEditKey('unknown')).toBeNull();
	});

	it('reports editability from the stored token', () => {
		expect(canEditMap('mine')).toBe(false);
		setMapEditKey('mine', 'token-a');
		expect(canEditMap('mine')).toBe(true);
	});

	it('forgets a token after deletion', () => {
		setMapEditKey('mine', 'token-a');
		forgetMapEditKey('mine');
		expect(getMapEditKey('mine')).toBeNull();
		expect(canEditMap('mine')).toBe(false);
	});

	it('ignores empty values', () => {
		setMapEditKey('mine', '');
		expect(getMapEditKey('mine')).toBeNull();
	});

	it('survives corrupted storage', () => {
		window.localStorage.setItem('tesla-road-trip.mapEditKeys', 'not json');
		expect(getMapEditKey('mine')).toBeNull();
		// and can still write afterwards
		setMapEditKey('mine', 'token-a');
		expect(getMapEditKey('mine')).toBe('token-a');
	});

	it('passes actionable authorization errors through', () => {
		expect(mapSaveErrorMessage(new Error('edit token required to modify this map'))).toContain('edit token');
		expect(mapSaveErrorMessage(new Error('invalid edit token'))).toContain('invalid edit token');
		expect(mapSaveErrorMessage(new Error('map already exists: "mine"'))).toContain('already exists');
		expect(mapSaveErrorMessage(new Error('forbidden: built-in maps can only be modified with an admin key'))).toContain(
			'built-in'
		);
	});
});
