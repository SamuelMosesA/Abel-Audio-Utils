<script lang="ts">
    import "../app.css";
    import { onMount } from "svelte";
    import { AppState, setAppContext } from "$lib/audioState.svelte";
    import NotificationBanner from "$lib/components/ui/NotificationBanner.svelte";

    let { children } = $props();
    const appState = new AppState();
    setAppContext(appState);

    onMount(() => {
        const reportError = async (message: string, stack: string) => {
            try {
                await fetch('/api/telemetry/errors', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ message, stack, url: window.location.href })
                });
            } catch (e) {
                console.error('Failed to report error:', e);
            }
        };

        const handleErrorEvent = (event: ErrorEvent) => {
            reportError(event.message, event.error?.stack || '');
        };

        const handleRejectionEvent = (event: PromiseRejectionEvent) => {
            const reason = event.reason;
            const message = reason instanceof Error ? reason.message : String(reason);
            const stack = reason instanceof Error ? reason.stack || '' : '';
            reportError(`Unhandled Promise Rejection: ${message}`, stack);
        };

        window.addEventListener('error', handleErrorEvent);
        window.addEventListener('unhandledrejection', handleRejectionEvent);

        return () => {
            window.removeEventListener('error', handleErrorEvent);
            window.removeEventListener('unhandledrejection', handleRejectionEvent);
        };
    });
</script>

<NotificationBanner />

<div class="min-h-screen flex flex-col bg-background text-foreground antialiased selection:bg-primary/20 selection:text-primary">
    <header class="sticky top-0 z-40 w-full border-b border-border/40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div class="max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8 flex h-14 items-center justify-between">
            <a href="/" class="flex items-center gap-2.5 font-semibold text-foreground tracking-tight hover:opacity-90 transition-opacity">
                <div class="w-7 h-7 rounded-md bg-primary/15 border border-primary/20 flex items-center justify-center font-bold text-xs text-primary">
                    AV
                </div>
                <span class="text-sm font-semibold tracking-tight">Abel Audio</span>
            </a>
            <nav class="flex items-center gap-3">
                <a href="/login" class="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors px-3 py-1.5 rounded-md hover:bg-muted/50 border border-transparent hover:border-border/40">
                    Admin
                </a>
            </nav>
        </div>
    </header>

    <main class="flex-1 w-full max-w-screen-xl mx-auto px-4 sm:px-6 lg:px-8 py-8 md:py-10">
        {@render children()}
    </main>

    <footer class="border-t border-border/40 py-6 text-center text-xs text-muted-foreground">
        <div class="max-w-screen-xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-4">
            <p>Abel Audio Proxy &copy; 2026</p>
            <p class="text-muted-foreground/60">Low Latency Audio & AI Broadcast</p>
        </div>
    </footer>
</div>
