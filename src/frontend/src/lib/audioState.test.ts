import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AudioStore, AIStore, SystemStore, UIStore, FileStore } from './audioState.svelte';
import { resolveRedirect } from './utils/redirect';

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

    describe('SystemStore', () => {
        it('should send DELETE /api/auth/session on logout and reset state', async () => {
            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const system = new SystemStore(ui, audio, ai);
            system.isAuthenticated = true;
            system.sessionId = "session-test-id";

            (fetch as any).mockResolvedValueOnce({
                ok: true,
                json: async () => ({ status: 'logged_out' })
            });

            await system.logout();

            expect(fetch).toHaveBeenCalledWith('/api/auth/session', expect.objectContaining({
                method: 'DELETE',
                credentials: 'include'
            }));
            expect(system.isAuthenticated).toBe(false);
            expect(system.sessionId).toBe("");
        });

        it("should validate active session with GET /api/auth/session", async () => {
            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const system = new SystemStore(ui, audio, ai);

            (fetch as any).mockResolvedValueOnce({
                ok: true,
                json: async () => ({ status: "authenticated", session: "validated-token", username: "admin" })
            });

            const isValid = await system.validateSession();

            expect(isValid).toBe(true);
            expect(system.isAuthenticated).toBe(true);
            expect(system.sessionId).toBe("validated-token");
        });

        it("should clear session and return false when session check fails", async () => {
            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const system = new SystemStore(ui, audio, ai);
            system.isAuthenticated = true;
            system.sessionId = "old-stale-token";

            (fetch as any).mockResolvedValueOnce({
                ok: false,
                status: 401,
                json: async () => ({ error: "Unauthorized session" })
            });

            const isValid = await system.validateSession();

            expect(isValid).toBe(false);
            expect(system.isAuthenticated).toBe(false);
            expect(system.sessionId).toBe("");
        });

        it("should not assume authentication on load even if localStorage has token", () => {
            localStorage.setItem("session_id", "some-stored-token");
            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const system = new SystemStore(ui, audio, ai);

            expect(system.isAuthenticated).toBe(false);
            expect(system.sessionId).toBe("some-stored-token");
        });

        it('should trigger files.fetchFiles and audio.sync on recording section update', async () => {
            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const files = new FileStore();
            const system = new SystemStore(ui, audio, ai, files);

            const audioSyncSpy = vi.spyOn(audio, 'sync').mockResolvedValue(undefined as any);
            const filesFetchSpy = vi.spyOn(files, 'fetchFiles').mockResolvedValue(undefined as any);

            system.handleRemoteUpdate({ section: 'recording', sessionId: 'other-session' });

            expect(audioSyncSpy).toHaveBeenCalled();
            expect(filesFetchSpy).toHaveBeenCalled();
        });
    });

    describe('Navigation and Redirects', () => {
        it('should resolve safe internal redirects', () => {
            expect(resolveRedirect('/admin')).toBe('/admin');
            expect(resolveRedirect('/admin/settings')).toBe('/admin/settings');
            expect(resolveRedirect('  /admin  ')).toBe('/admin');
        });

        it('should fallback to /admin on missing or empty redirect', () => {
            expect(resolveRedirect(null)).toBe('/admin');
            expect(resolveRedirect(undefined)).toBe('/admin');
            expect(resolveRedirect('')).toBe('/admin');
        });

        it('should prevent open redirect attacks', () => {
            expect(resolveRedirect('https://evil.com')).toBe('/admin');
            expect(resolveRedirect('//evil.com')).toBe('/admin');
            expect(resolveRedirect('/\\evil.com')).toBe('/admin');
            expect(resolveRedirect('javascript:alert(1)')).toBe('/admin');
        });
    });
});
