import { describe, expect, it } from 'vitest';
import { buildEditWithAIPrompt } from './editWithAI';

const map = { mapId: 'canyon_30', name: 'Canyon 30', gridSize: 30, maxBattery: 50 };

describe('buildEditWithAIPrompt', () => {
	it('includes the endpoint, map id and edit key', () => {
		const prompt = buildEditWithAIPrompt({
			map,
			editKey: 'SECRET-KEY-123',
			apiUrl: 'https://tesla.example.com/graphql'
		});

		expect(prompt).toContain('https://tesla.example.com/graphql');
		expect(prompt).toContain('canyon_30');
		expect(prompt).toContain('Canyon 30');
		expect(prompt).toContain('SECRET-KEY-123');
	});

	it('states the grid and battery so the agent can build a valid layout', () => {
		const prompt = buildEditWithAIPrompt({ map, editKey: 'k', apiUrl: 'http://x/graphql' });
		expect(prompt).toContain('30x30');
		expect(prompt).toMatch(/exactly 30 rows of 30 characters/);
		expect(prompt).toContain('max 50');
	});

	it('names the mutation and the authorization argument', () => {
		const prompt = buildEditWithAIPrompt({ map, editKey: 'k', apiUrl: 'http://x/graphql' });
		expect(prompt).toContain('updateMap');
		expect(prompt).toContain('editKey');
		// The token goes into the variables block, not just the prose.
		expect(prompt).toMatch(/"editKey":\s*"k"/);
	});

	it('tells the agent to read the map first, without credentials', () => {
		const prompt = buildEditWithAIPrompt({ map, editKey: 'k', apiUrl: 'http://x/graphql' });
		expect(prompt).toContain('query GetMap');
		expect(prompt).toMatch(/requires no credentials/);
	});

	it('degrades to asking for the key when this browser has none', () => {
		const prompt = buildEditWithAIPrompt({ map, editKey: null, apiUrl: 'http://x/graphql' });
		expect(prompt).toContain('I do not have the edit key');
		expect(prompt).toContain('<ASK_ME>');
		// It must not fabricate a token or emit an empty string as the key.
		expect(prompt).not.toMatch(/"editKey":\s*""/);
	});

	it('carries the layout rules the server enforces', () => {
		const prompt = buildEditWithAIPrompt({ map, editKey: 'k', apiUrl: 'http://x/graphql' });
		for (const rule of ['At least one H', 'at least one P', 'reachable', 'R road']) {
			expect(prompt).toContain(rule);
		}
	});

	it('falls back to the map id when the display name is empty', () => {
		const prompt = buildEditWithAIPrompt({
			map: { ...map, name: '' },
			editKey: 'k',
			apiUrl: 'http://x/graphql'
		});
		expect(prompt).toContain('name:   canyon_30');
	});
});
