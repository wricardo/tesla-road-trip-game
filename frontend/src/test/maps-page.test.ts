import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MapsPage from '../routes/maps/+page.svelte';

const runtime = vi.hoisted(() => ({
	mutations: [] as Array<{ query: string; variables: Record<string, unknown> }>,
	mapId: 'classic',
	lastSessionId: 'session-1'
}));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('@urql/svelte', () => ({
	gql: (query: string) => query,
	getContextClient: () => ({
		mutation: (query: string, variables: Record<string, unknown>) => ({
			toPromise: async () => {
				runtime.mutations.push({ query, variables });
				return { data: { createSession: { id: runtime.lastSessionId } } };
			}
		}),
		query: () => ({ toPromise: async () => ({ data: { map: null } }) })
	}),
	queryStore: () => ({
		subscribe: (run: (value: unknown) => void) => {
			run({
				data: {
					maps: [{ mapId: runtime.mapId, name: 'Classic Layout', description: 'The original', gridSize: 15, maxBattery: 20 }]
				}
			});
			return () => {};
		},
		reexecute: vi.fn()
	})
}));

describe('maps page', () => {
	beforeEach(() => {
		runtime.mutations = [];
		runtime.mapId = 'classic';
	});

	afterEach(() => {
		cleanup();
		vi.clearAllMocks();
	});

	it('shows no Edit or Duplicate actions on map cards', async () => {
		render(MapsPage);

		await screen.findByText('Classic Layout');
		expect(screen.queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument();
		expect(screen.queryByRole('link', { name: 'Duplicate' })).not.toBeInTheDocument();
		expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument();
		expect(screen.queryByRole('button', { name: 'Duplicate' })).not.toBeInTheDocument();
		// Still reachable from the page header for authoring a map.
		expect(screen.getByRole('link', { name: /Design a map/ })).toHaveAttribute('href', '/editor');
	});

	it('opens the settings modal on Play instead of creating a session immediately', async () => {
		render(MapsPage);

		await fireEvent.click(await screen.findByRole('button', { name: 'Play' }));

		expect(await screen.findByRole('dialog')).toBeInTheDocument();
		expect(screen.getByText('Start Classic Layout')).toBeInTheDocument();
		// Nothing is created until the player confirms in the modal.
		expect(runtime.mutations).toHaveLength(0);
	});

	it('creates the session with fog enabled and radius 1 by default', async () => {
		render(MapsPage);

		await fireEvent.click(await screen.findByRole('button', { name: 'Play' }));
		await fireEvent.click(await screen.findByRole('button', { name: 'Start session' }));

		await waitFor(() => expect(runtime.mutations).toHaveLength(1));
		expect(runtime.mutations[0].variables).toMatchObject({
			mapID: 'classic',
			fogEnabled: true,
			fogRadius: 1,
			gridPassword: null,
			moveDelayMs: 300
		});
	});

	it('passes edited settings from the modal to the session', async () => {
		render(MapsPage);

		await fireEvent.click(await screen.findByRole('button', { name: 'Play' }));
		await fireEvent.input(await screen.findByLabelText('Radius'), { target: { value: '4' } });
		await fireEvent.input(screen.getByLabelText('Move delay (ms)'), { target: { value: '0' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Start session' }));

		await waitFor(() => expect(runtime.mutations).toHaveLength(1));
		expect(runtime.mutations[0].variables).toMatchObject({ fogEnabled: true, fogRadius: 4, moveDelayMs: 0 });
	});
});
