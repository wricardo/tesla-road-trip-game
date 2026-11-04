import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import StartSessionModal from '$lib/components/StartSessionModal.svelte';
import { DEFAULT_MOVE_DELAY_MS } from '$lib/createSession';

const map = { mapId: 'classic', name: 'Classic Layout', description: '', gridSize: 15, maxBattery: 20 };

function renderModal(overrides: { starting?: boolean; error?: string | null } = {}) {
	const onstart = vi.fn();
	const onclose = vi.fn();
	const result = render(StartSessionModal, {
		map,
		starting: overrides.starting ?? false,
		error: overrides.error ?? null,
		onstart,
		onclose
	});
	return { ...result, onstart, onclose };
}

describe('StartSessionModal', () => {
	afterEach(() => cleanup());

	it('opens with fog enabled at radius 1 and the default move delay', () => {
		renderModal();

		expect(screen.getByRole('dialog')).toBeInTheDocument();
		expect(screen.getByLabelText('Hide the map beyond a small radius')).toBeChecked();
		expect(screen.getByLabelText('Radius')).toHaveValue(1);
		expect(screen.getByLabelText('Move delay (ms)')).toHaveValue(DEFAULT_MOVE_DELAY_MS);
	});

	it('submits the default settings unchanged', async () => {
		const { onstart } = renderModal();

		await fireEvent.click(screen.getByRole('button', { name: 'Start session' }));

		expect(onstart).toHaveBeenCalledWith({
			fogEnabled: true,
			fogRadius: 1,
			gridPassword: '',
			moveDelayMs: DEFAULT_MOVE_DELAY_MS
		});
	});

	it('submits edited fog and delay values', async () => {
		const { onstart } = renderModal();

		await fireEvent.input(screen.getByLabelText('Radius'), { target: { value: '3' } });
		await fireEvent.input(screen.getByLabelText('Move delay (ms)'), { target: { value: '750' } });
		await fireEvent.input(screen.getByLabelText('Grid password'), { target: { value: 'secret' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Start session' }));

		expect(onstart).toHaveBeenCalledWith({
			fogEnabled: true,
			fogRadius: 3,
			gridPassword: 'secret',
			moveDelayMs: 750
		});
	});

	it('hides the radius and password fields when fog is turned off', async () => {
		const { onstart } = renderModal();

		await fireEvent.click(screen.getByLabelText('Hide the map beyond a small radius'));
		expect(screen.queryByLabelText('Radius')).not.toBeInTheDocument();

		await fireEvent.click(screen.getByRole('button', { name: 'Start session' }));
		expect(onstart).toHaveBeenCalledWith(expect.objectContaining({ fogEnabled: false }));
	});

	it('blocks submission and explains an invalid radius', async () => {
		const { onstart } = renderModal();

		await fireEvent.input(screen.getByLabelText('Radius'), { target: { value: '0' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Start session' }));

		expect(onstart).not.toHaveBeenCalled();
		expect(screen.getByRole('alert')).toHaveTextContent('at least 1');
	});

	it('shows a server error and disables the submit button while starting', () => {
		renderModal({ starting: true, error: 'Could not create the session' });

		expect(screen.getByRole('alert')).toHaveTextContent('Could not create the session');
		expect(screen.getByRole('button', { name: 'Starting…' })).toBeDisabled();
	});

	it('closes on cancel and on Escape', async () => {
		const { onclose } = renderModal();

		await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
		expect(onclose).toHaveBeenCalledTimes(1);

		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onclose).toHaveBeenCalledTimes(2);
	});
});
