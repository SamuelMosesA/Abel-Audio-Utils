import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AudioStore, AIStore, SystemStore, UIStore } from './audioState.svelte';


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
        it('stale localStorage.session_id does not set isAuthenticated when session validation returns 401, and stale values are removed', async () => {
            localStorage.setItem("session_id", "stale-session-token");
            localStorage.setItem("admin_user", "stale-user");

            (fetch as any).mockImplementation(async (input: any) => {
                const url = typeof input === 'string' ? input : input.url;
                if (url === '/api/auth/session') {
                    return {
                        ok: false,
                        status: 401,
                        json: async () => ({ authenticated: false, error: 'Unauthorized session' })
                    };
                }
                return { ok: true, json: async () => ({}) };
            });

            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const system = new SystemStore(ui, audio, ai);

            // Initially before validation finishes, isAuthenticated must not be true from localStorage
            expect(system.isAuthenticated).toBe(false);

            await system.validateSession();

            expect(system.isAuthenticated).toBe(false);
            expect(system.sessionId).toBe("");
            expect(system.authChecked).toBe(true);
            expect(localStorage.getItem("session_id")).toBeNull();
            expect(localStorage.getItem("admin_user")).toBeNull();
        });

        it('should authenticate and set sessionId when session validation returns 200', async () => {
            (fetch as any).mockImplementation(async (input: any) => {
                const url = typeof input === 'string' ? input : input.url;
                if (url === '/api/auth/session') {
                    return {
                        ok: true,
                        status: 200,
                        json: async () => ({ authenticated: true, session_id: 'valid-session-xyz', username: 'admin' })
                    };
                }
                if (url === '/api/system/connection') {
                    return { ok: true, json: async () => ({ serverUrl: 'http://localhost', ssid: 'Abel' }) };
                }
                return { ok: true, json: async () => ({}) };
            });

            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const system = new SystemStore(ui, audio, ai);

            await system.validateSession();

            expect(system.isAuthenticated).toBe(true);
            expect(system.sessionId).toBe("valid-session-xyz");
            expect(system.authChecked).toBe(true);
            expect(localStorage.getItem("session_id")).toBe("valid-session-xyz");
        });

        it('should call DELETE /api/auth/session and clear state on logout', async () => {
            localStorage.setItem("session_id", "active-session");
            localStorage.setItem("admin_user", "admin");

            (fetch as any).mockImplementation(async (input: any, init?: any) => {
                const url = typeof input === 'string' ? input : input.url;
                const method = (init?.method || 'GET').toUpperCase();
                if (url === '/api/auth/session' && method === 'DELETE') {
                    return { ok: true, status: 200, json: async () => ({ status: 'logged_out' }) };
                }
                return { ok: true, json: async () => ({}) };
            });

            const ui = new UIStore();
            const audio = new AudioStore();
            const ai = new AIStore(ui);
            const system = new SystemStore(ui, audio, ai);
            system.isAuthenticated = true;
            system.sessionId = "active-session";

            await system.logout();

            expect(fetch).toHaveBeenCalledWith('/api/auth/session', expect.objectContaining({ method: 'DELETE' }));
            expect(system.isAuthenticated).toBe(false);
            expect(system.sessionId).toBe("");
            expect(localStorage.getItem("session_id")).toBeNull();
            expect(localStorage.getItem("admin_user")).toBeNull();
        });
    });
});
