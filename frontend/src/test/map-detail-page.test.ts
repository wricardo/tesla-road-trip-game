import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MapDetailPage from '../routes/maps/[id]/+page.svelte';

const runtime = vi.hoisted(() => ({
	mapId: 'classic',
	mutations: [] as Array<{ query: string; variables: Record<string, unknown> }>,
	layoutFound: true,
	maps: [
		{ mapId: 'classic', name: 'Classic Layout', description: 'The original layout', gridSize: 5, maxBattery: 20 },
		{ mapId: 'easy', name: 'Easy Mode', description: 'Beginner friendly', gridSize: 10, maxBattery: 15 }
	] as Array<Record<string, unknown>>
}));

const LAYOUT = {
	name: 'Classic Layout',
	description: 'The original layout',
	gridSize: 5,
	maxBattery: 20,
	startingBattery: 20,
	layout: ['BBBBB', 'BRHPB', 'BRRSB', 'BPPPB', 'BBBBB'],
	legend: [
		{ key: 'R', value: 'road' },
		{ key: 'H', value: 'home' },
		{ key: 'P', value: 'park' },
		{ key: 'S', value: 'supercharger' },
		{ key: 'W', value: 'water' },
		{ key: 'B', value: 'building' }
	],
	cellConfigs: []
};

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('$app/stores', async () => {
	// svelte/store is only imported here because vi.mock factories are hoisted
	// above the module's static imports; this module path is known at author time.
	const { readable } = await import('svelte/store');
	// Rebuild on subscribe so tests can change runtime.mapId between cases.
	type PageStore = { params: { id: string }; url: URL };
	return {
		page: readable<PageStore>(
			{ params: { id: runtime.mapId }, url: new URL('http://localhost:8000/maps/' + runtime.mapId) },
			(set) => {
				set({
					params: { id: runtime.mapId },
					url: new URL('http://localhost:8000/maps/' + runtime.mapId)
				});
			}
		)
	};
});

vi.mock('@urql/svelte', () => ({
	gql: (query: string) => query,
	getContextClient: () => ({
		mutation: (query: string, variables: Record<string, unknown>) => ({
			toPromise: async () => {
				runtime.mutations.push({ query, variables });
				return { data: { createSession: { id: 'session-42' } } };
			}
		}),
		query: () => ({
			toPromise: async () => ({ data: { map: runtime.layoutFound ? LAYOUT : null } })
		})
	}),
	queryStore: () => ({
		subscribe: (run: (value: unknown) => void) => {
			run({ data: { maps: runtime.maps } });
			return () => {};
		},
		reexecute: vi.fn()
	})
}));

describe('map detail page', () => {
	beforeEach(() => {
		runtime.mapId = 'classic';
		runtime.mutations = [];
		runtime.layoutFound = true;
		window.localStorage.clear();
	});

	afterEach(() => {
		cleanup();
		vi.clearAllMocks();
	});

	it('shows the map name, description and layout', async () => {
		render(MapDetailPage);

		expect(await screen.findByRole('heading', { level: 1, name: 'Classic Layout' })).toBeInTheDocument();
		expect(await screen.findByText('The original layout')).toBeInTheDocument();
		// The layout is fetched in an effect, so wait for it before asserting.
		expect(await screen.findByRole('img', { name: 'Layout of Classic Layout' })).toBeInTheDocument();
		// Battery is shown as starting / max.
		expect(screen.getByText('20 / 20')).toBeInTheDocument();
	});

	it('counts parks, chargers, homes and obstacles from the layout', async () => {
		render(MapDetailPage);

		await screen.findByRole('img', { name: 'Layout of Classic Layout' });
		// Layout rows: "BBBBB", "BRHPB", "BRRSB", "BPPPB", "BBBBB".
		// Parks 1+0+3 = 4, chargers 1, homes 1, obstacles = 16 buildings + 0 water.
		expect(await screen.findByText(/Parks/)).toBeInTheDocument();
		const stat = (label: string) => screen.getByText(new RegExp(`\\b${label}$`));
		expect(stat('Parks').parentElement).toHaveTextContent('4');
		expect(stat('Chargers').parentElement).toHaveTextContent('1');
		expect(stat('Homes').parentElement).toHaveTextContent('1');
		expect(stat('Obstacles').parentElement).toHaveTextContent('16');
	});

	it('opens the start-session modal and creates a session for this map', async () => {
		render(MapDetailPage);

		await fireEvent.click(await screen.findByRole('button', { name: 'Play this map' }));

		expect(await screen.findByRole('dialog')).toBeInTheDocument();
		expect(screen.getByText('Start Classic Layout')).toBeInTheDocument();
		// Nothing is created until the player confirms.
		expect(runtime.mutations).toHaveLength(0);

		await fireEvent.click(screen.getByRole('button', { name: 'Start session' }));

		await waitFor(() => expect(runtime.mutations).toHaveLength(1));
		expect(runtime.mutations[0].variables).toMatchObject({
			mapID: 'classic',
			fogEnabled: true,
			fogRadius: 1,
			moveDelayMs: 300
		});
	});

	it('links back to the map gallery', async () => {
		render(MapDetailPage);

		expect(await screen.findByRole('link', { name: /All maps/ })).toHaveAttribute('href', '/maps');
	});

	it('explains that an unknown map does not exist', async () => {
		runtime.mapId = 'nope';
		runtime.layoutFound = false;
		render(MapDetailPage);

		expect(await screen.findByRole('heading', { level: 1, name: 'Map not found' })).toBeInTheDocument();
		expect(screen.getByText('nope')).toBeInTheDocument();
		expect(screen.queryByRole('button', { name: 'Play this map' })).not.toBeInTheDocument();
	});

	it('offers to copy the page link for sharing', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		vi.stubGlobal('navigator', { clipboard: { writeText } });

		render(MapDetailPage);
		const [copy] = await screen.findAllByRole('button', { name: 'Copy link' });
		await fireEvent.click(copy);

		expect(writeText).toHaveBeenCalledWith('http://localhost:8000/maps/classic');
		// The button reflects the copied state (header and share panel both do).
		expect((await screen.findAllByText('Link copied ✓')).length).toBeGreaterThan(0);
		vi.unstubAllGlobals();
	});
});

describe('map detail page — editing affordances', () => {
	beforeEach(() => {
		runtime.mapId = 'classic';
		runtime.mutations = [];
		runtime.layoutFound = true;
		window.localStorage.clear();
	});

	afterEach(() => {
		cleanup();
		vi.clearAllMocks();
	});

	it('shows the stored edit key on the page, not only in localStorage', async () => {
		window.localStorage.setItem(
			'tesla-road-trip.mapEditKeys',
			JSON.stringify({ classic: 'KEY-ABC-123' })
		);
		render(MapDetailPage);

		const section = await screen.findByTestId('edit-key-section');
		expect(section).toHaveTextContent('Edit key');
		expect(screen.getByTestId('edit-key-value')).toHaveTextContent('KEY-ABC-123');
	});

	it('explains when this browser has no key for the map', async () => {
		render(MapDetailPage);
		const section = await screen.findByTestId('edit-key-section');
		expect(section).toHaveTextContent('no edit key');
		expect(screen.queryByTestId('edit-key-value')).not.toBeInTheDocument();
	});

	it('offers an AI prompt that embeds the key and the GraphQL endpoint', async () => {
		window.localStorage.setItem(
			'tesla-road-trip.mapEditKeys',
			JSON.stringify({ classic: 'KEY-ABC-123' })
		);
		render(MapDetailPage);

		const section = await screen.findByTestId('edit-with-ai-section');
		expect(section).toHaveTextContent('Edit with AI');

		const prompt = screen.getByTestId('ai-prompt-preview').textContent ?? '';
		expect(prompt).toContain('KEY-ABC-123');
		expect(prompt).toContain('classic');
		// The UI talks to a relative endpoint; the prompt must be absolute.
		// jsdom serves the page from http://localhost:3000.
		expect(prompt).toMatch(/^I want to change a map[\s\S]*http:\/\/[^\s]+\/graphql/);
		expect(prompt).toContain('updateMap');
	});

	it('copies the AI prompt to the clipboard', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		vi.stubGlobal('navigator', { clipboard: { writeText } });
		window.localStorage.setItem(
			'tesla-road-trip.mapEditKeys',
			JSON.stringify({ classic: 'KEY-ABC-123' })
		);
		render(MapDetailPage);

		await screen.findByTestId('edit-with-ai-section');
		await fireEvent.click(screen.getByRole('button', { name: 'Copy AI prompt' }));

		expect(writeText).toHaveBeenCalledTimes(1);
		const copied = writeText.mock.calls[0][0] as string;
		expect(copied).toContain('KEY-ABC-123');
		expect(copied).toContain('/graphql');
		expect(await screen.findByText('Prompt copied ✓')).toBeInTheDocument();
		vi.unstubAllGlobals();
	});
});
