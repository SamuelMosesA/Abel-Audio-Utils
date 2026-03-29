import { vi } from 'vitest';
import '@testing-library/jest-dom/vitest';

// Mock global fetch
global.fetch = vi.fn().mockImplementation((input: any) => {
    const url = typeof input === 'string' ? input : input.url;
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
    constructor(url: string, init?: any) {}
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
    constructor(url: string) {}
}
global.WebSocket = MockWebSocket as any;

global.alert = vi.fn();
