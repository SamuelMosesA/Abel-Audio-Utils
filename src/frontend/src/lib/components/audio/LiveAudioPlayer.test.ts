import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import LiveAudioPlayer from './LiveAudioPlayer.svelte';

describe('LiveAudioPlayer', () => {
    const play = vi.fn<() => Promise<void>>();

    beforeEach(() => {
        play.mockReset();
        Object.defineProperty(HTMLMediaElement.prototype, 'play', {
            configurable: true,
            value: play
        });
    });

    it('starts playback from an explicit user action', async () => {
        play.mockResolvedValueOnce();
        render(LiveAudioPlayer, { src: '/api/audio/stream' });

        await fireEvent.click(screen.getByRole('button', { name: 'Start Listening' }));

        expect(play).toHaveBeenCalledOnce();
    });

    it('surfaces a rejected playback request and allows retry', async () => {
        play.mockRejectedValueOnce(new DOMException('Not allowed', 'NotAllowedError'));
        render(LiveAudioPlayer, { src: '/api/audio/stream' });

        await fireEvent.click(screen.getByRole('button', { name: 'Start Listening' }));

        await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Browser blocked playback'));
        expect(screen.getByRole('button', { name: 'Retry Listening' })).toBeInTheDocument();
    });
});
