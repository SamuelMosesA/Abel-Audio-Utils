import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AudioStore, AIStore, SystemStore, UIStore } from './audioState.svelte';

// Mock fetch
global.fetch = vi.fn();

describe('Modular Stores', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe('AudioStore', () => {
        it('should fetch devices', async () => {
            const devices = [{ id: 0, name: 'Default', inputs: 2 }];
            (fetch as any).mockResolvedValueOnce({
                ok: true,
                json: async () => devices
            });

            const store = new AudioStore();
            await store.fetchDevices();
            expect(store.devices).toEqual(devices);
        });

        it('should commit config', async () => {
            (fetch as any).mockResolvedValue({ ok: true });
            const store = new AudioStore();
            store.chL = 5;
            await store.commitConfig(1);
            expect(fetch).toHaveBeenCalledWith('/api/audio/config', expect.objectContaining({
                method: 'PATCH',
                body: expect.stringContaining('"chL":5')
            }));
        });
    });

    describe('AIStore', () => {
        it('should toggle gemini master with correct casing', async () => {
            const ui = new UIStore();
            const store = new AIStore(ui);
            (fetch as any).mockResolvedValue({ 
                ok: true,
                json: async () => ({})
            });
            
            await store.setGeminiMaster(true);
            
            expect(fetch).toHaveBeenCalledWith('/api/ai/streams', expect.objectContaining({
                method: 'POST',
                body: expect.stringContaining('"action":"toggle_master","enabled":true')
            }));
        });
    });

    describe('UIStore', () => {
        it('should show notifications', () => {
            vi.useFakeTimers();
            const store = new UIStore();
            store.showNotification('Test Message', 'success');
            expect(store.notification).toEqual({ message: 'Test Message', section: 'success' });
            
            vi.advanceTimersByTime(5000);
            expect(store.notification).toBeNull();
            vi.useRealTimers();
        });
    });
});
