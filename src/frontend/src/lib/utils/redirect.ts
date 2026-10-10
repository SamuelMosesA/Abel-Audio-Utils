/**
 * Safely resolves an internal redirect destination, preventing open redirect vulnerabilities.
 */
export function resolveRedirect(target: string | null | undefined, fallback = "/admin"): string {
    if (!target) {
        return fallback;
    }
    const trimmed = target.trim();
    if (
        trimmed.startsWith("/") &&
        !trimmed.startsWith("//") &&
        !trimmed.startsWith("/\\") &&
        !trimmed.startsWith("/\t") &&
        !trimmed.startsWith("/\r") &&
        !trimmed.startsWith("/\n")
    ) {
        try {
            const parsed = new URL(trimmed, "http://localhost");
            if (parsed.origin === "http://localhost" && parsed.pathname.startsWith("/")) {
                return parsed.pathname + parsed.search + parsed.hash;
            }
        } catch {
            return fallback;
        }
    }
    return fallback;
}
