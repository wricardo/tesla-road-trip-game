/**
 * Edit tokens let a browser update or delete the maps it created without any
 * account system: the server returns the token once from createMap, and we keep
 * it locally. Losing it means the map can only be changed by an admin, so we
 * never silently drop an entry for a map that still exists.
 */
const STORAGE_KEY = 'tesla-road-trip.mapEditKeys';

type KeyMap = Record<string, string>;

/**
 * Storage lookup: prefer window.localStorage, which is also what jsdom/browsers
 * expose. (Node 2x ships a global `localStorage` that is inert unless a flag is
 * passed, so it must not be used directly in tests or SSR.)
 */
function storage(): Storage | null {
	if (typeof window === 'undefined') return null;
	try {
		return window.localStorage ?? null;
	} catch {
		// Access can throw when storage is disabled by policy.
		return null;
	}
}

function readAll(): KeyMap {
	const store = storage();
	if (!store) return {};
	try {
		const raw = store.getItem(STORAGE_KEY);
		if (!raw) return {};
		const parsed = JSON.parse(raw);
		return parsed && typeof parsed === 'object' ? (parsed as KeyMap) : {};
	} catch {
		return {};
	}
}

function writeAll(keys: KeyMap) {
	const store = storage();
	if (!store) return;
	try {
		store.setItem(STORAGE_KEY, JSON.stringify(keys));
	} catch {
		// Storage can be full or blocked (private mode); editing still works for
		// this session because the caller keeps the token in memory.
	}
}

/** Returns the stored edit token for a map, or null. */
export function getMapEditKey(mapId: string): string | null {
	return readAll()[mapId] ?? null;
}

/** Stores the edit token returned by createMap. */
export function setMapEditKey(mapId: string, editKey: string) {
	if (!mapId || !editKey) return;
	const keys = readAll();
	keys[mapId] = editKey;
	writeAll(keys);
}

/** Forgets the token for a map (after it is deleted). */
export function forgetMapEditKey(mapId: string) {
	const keys = readAll();
	if (!(mapId in keys)) return;
	delete keys[mapId];
	writeAll(keys);
}

/** True when this browser holds a token for the map. */
export function canEditMap(mapId: string): boolean {
	return Boolean(getMapEditKey(mapId));
}

/**
 * Turns a GraphQL error into a message that tells the user what to do next.
 * The server's authorization errors are already actionable, so they pass through.
 */
export function mapSaveErrorMessage(error: unknown): string {
	const message = error instanceof Error ? error.message : String(error ?? 'Unknown error');
	if (/edit token|editKey|already exists|built-in/i.test(message)) return message;
	return message;
}
