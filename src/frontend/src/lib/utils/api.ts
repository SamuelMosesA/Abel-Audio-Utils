/**
 * Utility to perform an API call.
 * The individual stores are responsible for syncing their own state after calls.
 */
export async function fetchWithSync(url: string, options: RequestInit = {}) {
    const headers = new Headers(options.headers || {});
    
    const response = await fetch(url, {
        ...options,
        headers,
        credentials: "include"
    });
    
    if (!response.ok && response.status === 401) {
        console.warn("Unauthorized API call detected");
    }
    
    return response;
}
