/**
 * Utility to perform an API call.
 * The individual stores are responsible for syncing their own state after calls.
 */

let unauthorizedHandler: (() => void) | null = null;

export function setUnauthorizedHandler(handler: (() => void) | null) {
    unauthorizedHandler = handler;
}

export function getUnauthorizedHandler() {
    return unauthorizedHandler;
}

export async function fetchWithSync(url: string, options: RequestInit = {}) {
    const headers = new Headers(options.headers || {});
    
    const response = await fetch(url, {
        ...options,
        headers,
        credentials: "include"
    });
    
    if (!response.ok && response.status === 401) {
        console.warn("Unauthorized API call detected");
        if (unauthorizedHandler) {
            unauthorizedHandler();
        }
    }
    
    return response;
}
