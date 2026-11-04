/**
 * Settings the play modal collects before a session is created.
 * Fog defaults to enabled with radius 1 so sessions start "blind" for AI agents.
 */
import { DEFAULT_MOVE_DELAY_MS } from '$lib/createSession';

export type StartSettings = {
	fogEnabled: boolean;
	fogRadius: number;
	gridPassword: string;
	moveDelayMs: number;
};

export const DEFAULT_START_SETTINGS: StartSettings = {
	fogEnabled: true,
	fogRadius: 1,
	gridPassword: '',
	moveDelayMs: DEFAULT_MOVE_DELAY_MS
};

/** Validates the form; returns an error message or null when the settings are usable. */
export function validateStartSettings(settings: StartSettings): string | null {
	if (!Number.isInteger(settings.fogRadius) || settings.fogRadius < 1) {
		return 'Fog radius must be a whole number of at least 1';
	}
	if (!Number.isFinite(settings.moveDelayMs) || settings.moveDelayMs < 0) {
		return 'Move delay must be 0 or greater';
	}
	return null;
}
