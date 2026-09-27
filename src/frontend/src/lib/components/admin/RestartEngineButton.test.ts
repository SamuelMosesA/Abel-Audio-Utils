import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import RestartEngineButton from './RestartEngineButton.svelte';

describe('RestartEngineButton', () => {
    it('is disabled while recording', () => {
        render(RestartEngineButton, { isRecording: true, isRestarting: false, onrestart: vi.fn() });

        expect(screen.getByRole('button', { name: 'Restart Engine' })).toBeDisabled();
        expect(screen.getByText('Stop recording to restart the engine.')).toBeInTheDocument();
    });

    it('is enabled when not recording and calls onrestart', async () => {
        const onrestart = vi.fn();
        render(RestartEngineButton, { isRecording: false, isRestarting: false, onrestart });

        const button = screen.getByRole('button', { name: 'Restart Engine' });
        expect(button).toBeEnabled();
        await fireEvent.click(button);
        expect(onrestart).toHaveBeenCalledOnce();
    });

    it('is disabled while a restart is in progress', () => {
        render(RestartEngineButton, { isRecording: false, isRestarting: true, onrestart: vi.fn() });

        expect(screen.getByRole('button', { name: 'Restarting Engine...' })).toBeDisabled();
    });
});
