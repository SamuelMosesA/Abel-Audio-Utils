import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AudioVisuals } from './audioVisuals.svelte';
import { UIStore, AudioStore, AIStore, SystemStore } from './audioState.svelte';

// Mock Global dependencies
global.requestAnimationFrame = vi.fn().mockImplementation((cb) => setTimeout(() => cb(Date.now()), 16));
global.AudioContext = class {
    state = 'suspended';
    currentTime = 0;
    resume = vi.fn().mockResolvedValue(undefined);
    suspend = vi.fn().mockResolvedValue(undefined);
    createBuffer = vi.fn().mockReturnValue({
        duration: 1,
        getChannelData: vi.fn().mockReturnValue(new Float32Array(1024))
    });
    createBufferSource = vi.fn().mockReturnValue({
        connect: vi.fn(),
        start: vi.fn(),
        buffer: null
    });
    destination = {};
} as any;

describe('AudioVisuals', () => {
    let visuals: AudioVisuals;

    beforeEach(() => {
        vi.useFakeTimers();
        const ui = new UIStore();
        const audio = new AudioStore();
        const ai = new AIStore(ui);
        const system = new SystemStore(ui, audio, ai);
        visuals = new AudioVisuals(system);
    });

    it('should process audio data and update dB values', async () => {
        const buffer = new ArrayBuffer(16);
        const dv = new DataView(buffer);
        dv.setFloat32(0, 0.5, true); // L (approx -6dB)
        dv.setFloat32(4, 0.1, true); // R (-20dB)

        visuals.processData(dv);

        // Advance timers multiple times to allow smooth transition
        for (let i = 0; i < 64; i++) {
            vi.advanceTimersByTime(16);
        }

        // dB values should be close to targets after some time
        expect(visuals.currentDb.L).toBeGreaterThan(-10);
        expect(visuals.currentDb.L).toBeLessThan(-3);
        expect(visuals.currentDb.R).toBeGreaterThan(-25);
        expect(visuals.currentDb.R).toBeLessThan(-15);
    });

    it('should toggle monitoring state', async () => {
        expect(visuals.monitoring).toBe(false);
        await visuals.toggleMonitor();
        expect(visuals.monitoring).toBe(true);
        
        await visuals.toggleMonitor();
        expect(visuals.monitoring).toBe(false);
    });
});
