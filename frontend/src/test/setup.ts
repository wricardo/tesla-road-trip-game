import '@testing-library/jest-dom/vitest';

/**
 * Node 2x defines a global `localStorage` that is inert unless started with
 * --localstorage-file, and that global shadows the one jsdom would install.
 * Provide a small in-memory Storage so tests can exercise code that persists
 * to localStorage (e.g. map edit tokens).
 */
if (typeof window !== 'undefined' && !window.localStorage) {
	const store = new Map<string, string>();
	const shim: Storage = {
		get length() {
			return store.size;
		},
		clear() {
			store.clear();
		},
		getItem(key: string) {
			return store.has(key) ? (store.get(key) as string) : null;
		},
		key(index: number) {
			return Array.from(store.keys())[index] ?? null;
		},
		removeItem(key: string) {
			store.delete(key);
		},
		setItem(key: string, value: string) {
			store.set(key, String(value));
		}
	};

	Object.defineProperty(window, 'localStorage', {value: shim, configurable: true});
	Object.defineProperty(globalThis, 'localStorage', {value: shim, configurable: true});
}
