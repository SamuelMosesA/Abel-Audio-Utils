import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AudioStore, AIStore, SystemStore, UIStore, describeRestart } from './audioState.svelte';


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
            (fetch as any).mockResolvedValue({ 
                ok: true,
                json: async () => ({})
            });
            const store = new AudioStore();
            store.chL = 5;
            await store.commitConfig(1);
            expect(fetch).toHaveBeenCalledWith('/api/audio/config', expect.objectContaining({
                method: 'PATCH',
                body: expect.stringContaining('"chL":5')
            }));
        });

        it('should restart the engine and refresh devices', async () => {
            const devices = [{ id: 0, name: 'Behringer UMC404HD', inputs: 4 }];
            (fetch as any)
                .mockResolvedValueOnce({
                    ok: true,
                    json: async () => ({ devices, deviceID: 0, restoredDevice: 'Behringer UMC404HD', configReloaded: true })
                })
                .mockResolvedValue({ ok: true, json: async () => ({ isRunning: true, deviceID: 0 }) });

            const store = new AudioStore();
            const result = await store.restartEngine();

            expect(fetch).toHaveBeenCalledWith('/api/system/restart', expect.objectContaining({ method: 'POST' }));
            expect(fetch).toHaveBeenCalledWith('/api/audio/config', expect.anything());
            expect(result).toEqual({ ok: true, message: 'Engine restarted. 1 device found. Reconnected to Behringer UMC404HD.' });
            expect(store.devices).toEqual(devices);
            expect(store.isRestarting).toBe(false);
        });

        it('should surface a refused restart', async () => {
            (fetch as any)
                .mockResolvedValueOnce({
                    ok: false,
                    status: 409,
                    json: async () => ({ error: 'cannot restart the engine while recording' })
                })
                .mockResolvedValue({ ok: true, json: async () => ({ isRecording: true }) });

            const store = new AudioStore();
            const result = await store.restartEngine();

            expect(result).toEqual({ ok: false, message: 'cannot restart the engine while recording' });
            expect(store.isRestarting).toBe(false);
        });
    });

    describe('describeRestart', () => {
        it('explains a missing device, config errors and settings needing a full restart', () => {
            expect(describeRestart({
                devices: [],
                deviceID: -1,
                missingDevice: 'Behringer UMC404HD',
                configReloaded: false,
                configError: 'yaml: bad indent',
                needsFullRestart: ['port']
            })).toBe(
                'Engine restarted. 0 devices found. Behringer UMC404HD could not be reconnected. Please select a device. ' +
                'Settings were not reloaded: yaml: bad indent Close and reopen Abel to apply: port.'
            );
        });
    });

    describe('AIStore', () => {
        it('should toggle AI master with correct casing', async () => {
            const ui = new UIStore();
            const store = new AIStore(ui);
            (fetch as any).mockResolvedValue({ 
                ok: true,
                json: async () => ({})
            });
            
            await store.setAIMaster(true);
            
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
