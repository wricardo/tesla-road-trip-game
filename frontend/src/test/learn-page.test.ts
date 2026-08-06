import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import LearnPage from '../routes/learn/+page.svelte';

describe('learn page copy snippets', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		Object.defineProperty(globalThis.navigator, 'clipboard', {
			value: {
				writeText: vi.fn().mockResolvedValue(undefined)
			},
			configurable: true
		});
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.restoreAllMocks();
	});

	it('copies snippet text and shows transient copied state', async () => {
		render(LearnPage);
		const initialCopyButtons = screen.getAllByRole('button', { name: 'Copy' }).length;

		const [copyButton] = screen.getAllByRole('button', { name: 'Copy' });
		await fireEvent.click(copyButton);

		expect(navigator.clipboard.writeText).toHaveBeenCalledTimes(1);
		expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument();

		await vi.advanceTimersByTimeAsync(1200);

		expect(screen.queryByRole('button', { name: 'Copied' })).not.toBeInTheDocument();
		expect(screen.getAllByRole('button', { name: 'Copy' })).toHaveLength(initialCopyButtons);
	});
});
