import { audioState } from "../audioState.svelte";

/**
 * Utility to perform an API call and immediately sync the application state.
 */
export async function fetchWithSync(url: string, options: RequestInit = {}) {
    const adminPassword = localStorage.getItem("admin_password") || "";
    
    const headers = new Headers(options.headers || {});
    if (adminPassword) {
        headers.set("X-Admin-Password", adminPassword);
    }
    
    const response = await fetch(url, {
        ...options,
        headers
    });
    
    if (response.ok) {
        await audioState.syncStatus();
    } else if (response.status === 401) {
        console.warn("Unauthorized API call, logging out...");
        audioState.logout();
    }
    
    return response;
}
