import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import AudioPlayer from './AudioPlayer.svelte';

describe('AudioPlayer', () => {
    it('keeps the verified MP3 duration when the browser reports a later end time', async () => {
        const { container } = render(AudioPlayer, {
            src: '/api/recordings/raw/processed.mp3',
            duration: 2898.2
        });
        const audio = container.querySelector('audio')!;
        const slider = screen.getByRole('slider', { name: 'Audio position' }) as HTMLInputElement;

        Object.defineProperty(audio, 'duration', { configurable: true, value: 2914.47 });
        Object.defineProperty(audio, 'currentTime', { configurable: true, value: 2914.47 });
        await fireEvent.loadedMetadata(audio);
        await fireEvent.timeUpdate(audio);

        expect(screen.getAllByText('48:18')).toHaveLength(2);
        expect(screen.queryByText('48:34')).not.toBeInTheDocument();
        expect(slider.max).toBe('2898.2');
        expect(Number(slider.value)).toBeLessThanOrEqual(2898.2);
    });
});
