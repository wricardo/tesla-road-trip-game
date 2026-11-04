import { type Client } from '@urql/svelte';
import { goto } from '$app/navigation';
import { createGameSession } from '$lib/createSession';
import type { MapSummary } from '$lib/maps';
import type { StartSettings } from '$lib/startSettings';

/** The subset of a map a session can be started from. */
export type StartableMap = Pick<MapSummary, 'mapId' | 'name' | 'description' | 'gridSize' | 'maxBattery'>;

/**
 * Owns the "press Play → choose fog/delay → create session" flow so the home
 * page, the map gallery and a single map's page all behave identically.
 */
export class MapSessionStarter {
	#client: Client;
	#onStarted: (sessionID: string) => void;

	map = $state<StartableMap | null>(null);
	starting = $state(false);
	error = $state('');

	constructor(client: Client, onStarted: (sessionID: string) => void = (id) => void goto(`/watch/${id}`)) {
		this.#client = client;
		this.#onStarted = onStarted;
	}

	/** Opens the settings modal for a map. */
	open(map: StartableMap) {
		if (this.starting) return;
		this.error = '';
		this.map = map;
	}

	/** Closes the modal, unless a session is being created. */
	cancel() {
		if (this.starting) return;
		this.map = null;
		this.error = '';
	}

	/** Creates the session with the chosen settings; keeps the modal open on failure. */
	async begin(settings: StartSettings, sessionName?: string) {
		const map = this.map;
		if (!map || this.starting) return;
		this.starting = true;
		this.error = '';
		try {
			const id = await createGameSession(this.#client, {
				mapID: map.mapId,
				name: sessionName,
				fogEnabled: settings.fogEnabled,
				fogRadius: settings.fogRadius,
				gridPassword: settings.gridPassword,
				moveDelayMs: settings.moveDelayMs
			});
			this.#onStarted(id);
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Could not create the session';
		} finally {
			this.starting = false;
		}
	}
}
