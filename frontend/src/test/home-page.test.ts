import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import HomePage from '../routes/+page.svelte';

const runtime = vi.hoisted(() => ({
	mutations: [] as Array<{ query: string; variables: Record<string, unknown> }>
}));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('@urql/svelte', () => ({
	gql: (query: string) => query,
	getContextClient: () => ({
		mutation: (query: string, variables: Record<string, unknown>) => ({
			toPromise: async () => {
				runtime.mutations.push({ query, variables });
				return { data: { createSession: { id: 'session-1' } } };
			}
		}),
		query: () => ({ toPromise: async () => ({ data: { map: null } }) })
	}),
	queryStore: ({ query }: { query: string }) => ({
		subscribe: (run: (value: unknown) => void) => {
			if (query.includes('query Sessions')) {
				run({ data: { sessions: { sessions: [] } } });
			} else {
				run({
					data: {
						maps: [{ mapId: 'classic', name: 'Classic Layout', description: 'The original', gridSize: 15, maxBattery: 20 }]
					}
				});
			}
			return () => {};
		},
		reexecute: vi.fn()
	})
}));

describe('home page start flow', () => {
	beforeEach(() => {
		runtime.mutations = [];
	});

	afterEach(() => {
		cleanup();
		vi.clearAllMocks();
	});

	it('no longer renders the inline Advanced options block', async () => {
		render(HomePage);

		await screen.findByText('Classic Layout');
		expect(screen.queryByText(/Advanced options/)).not.toBeInTheDocument();
		expect(screen.queryByLabelText('Move delay (ms)')).not.toBeInTheDocument();
	});

	it('opens the modal on Play and defaults to fog at radius 1', async () => {
		render(HomePage);

		await fireEvent.click(await screen.findByRole('button', { name: 'Play' }));

		expect(await screen.findByRole('dialog')).toBeInTheDocument();
		expect(screen.getByLabelText('Hide the map beyond a small radius')).toBeChecked();
		expect(screen.getByLabelText('Radius')).toHaveValue(1);
		expect(runtime.mutations).toHaveLength(0);
	});

	it('creates the session with fog enabled at radius 1 by default', async () => {
		render(HomePage);

		await fireEvent.click(await screen.findByRole('button', { name: 'Play' }));
		await fireEvent.click(await screen.findByRole('button', { name: 'Start session' }));

		await waitFor(() => expect(runtime.mutations.length).toBeGreaterThan(0));
		expect(runtime.mutations[0].variables).toMatchObject({
			mapID: 'classic',
			fogEnabled: true,
			fogRadius: 1,
			gridPassword: null,
			moveDelayMs: 300
		});
	});

	it('opens the modal from the empty-state "Start a session" button', async () => {
		render(HomePage);

		await fireEvent.click(await screen.findByRole('button', { name: 'Start a session' }));

		expect(await screen.findByRole('dialog')).toBeInTheDocument();
		expect(screen.getByText('Start Classic Layout')).toBeInTheDocument();
	});
});
