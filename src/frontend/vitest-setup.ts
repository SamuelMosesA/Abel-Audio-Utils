import { vi } from 'vitest';
import '@testing-library/jest-dom/vitest';

// Mock global fetch
global.fetch = vi.fn().mockImplementation((input: any, init?: any) => {
    const url = typeof input === 'string' ? input : input.url;
    const method = (init?.method || (typeof input === 'object' ? input.method : undefined) || 'GET').toUpperCase();
    if (url === '/api/auth/session') {
        if (method === 'DELETE') {
            return Promise.resolve({
                ok: true,
                status: 200,
                json: async () => ({ status: 'logged_out' }),
                text: async () => "",
                headers: new Headers(),
            });
        }
        return Promise.resolve({
            ok: false,
            status: 401,
            json: async () => ({ authenticated: false, error: 'Unauthorized session' }),
            text: async () => "",
            headers: new Headers(),
        });
    }
    // Handle relative URLs in Node/Vitest environment
    if (url && url.startsWith('/')) {
        return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ([]),
            text: async () => "",
            headers: new Headers(),
        });
    }
    return Promise.resolve({
        ok: true,
        json: async () => ({})
    });
});

// Mock global EventSource
class MockEventSource {
    close = vi.fn();
    onmessage = null;
    onerror = null;
    constructor(url: string, init?: any) { }
}
global.EventSource = MockEventSource as any;

// Mock global WebSocket
class MockWebSocket {
    static OPEN = 1;
    readyState = 1;
    close = vi.fn();
    send = vi.fn();
    onopen = null;
    onmessage = null;
    onclose = null;
    onerror = null;
    constructor(url: string) { }
}
global.WebSocket = MockWebSocket as any;

global.alert = vi.fn();

// Mock global localStorage
const localStorageMock = (() => {
    let store: Record<string, string> = {};
    return {
        getItem: (key: string) => store[key] || null,
        setItem: (key: string, value: string) => { store[key] = value.toString(); },
        removeItem: (key: string) => { delete store[key]; },
        clear: () => { store = {}; },
        key: (index: number) => Object.keys(store)[index] || null,
        get length() { return Object.keys(store).length; }
    };
})();
Object.defineProperty(global, 'localStorage', { value: localStorageMock });
