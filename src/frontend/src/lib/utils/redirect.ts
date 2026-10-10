/**
 * Safely resolves an internal redirect destination, preventing open redirect vulnerabilities.
 */
export function resolveRedirect(target: string | null | undefined, fallback = "/admin"): string {
    if (!target) {
        return fallback;
    }
    const trimmed = target.trim();
    if (trimmed.startsWith("/") && !trimmed.startsWith("//") && !trimmed.startsWith("/\\")) {
        return trimmed;
    }
    return fallback;
}
