<script lang="ts">
    import "../app.css";
    import { onMount } from "svelte";
    import { AppState, setAppContext } from "$lib/audioState.svelte";
    import { page } from "$app/state";
    import NotificationBanner from "$lib/components/ui/NotificationBanner.svelte";

    let { children } = $props();
    const appState = new AppState();
    setAppContext(appState);

    onMount(() => {
        // Initial setup
    });
</script>

<NotificationBanner />

<main class="min-h-screen bg-background text-foreground selection:bg-primary/30 selection:text-primary-foreground font-sans antialiased overflow-x-hidden">
    <!-- Ambient Background Elements -->
    <div class="fixed inset-0 overflow-hidden pointer-events-none z-0">
        <div class="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-primary/5 rounded-full blur-ultra"></div>
        <div class="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-amber-500/5 rounded-full blur-ultra"></div>
    </div>

    <div class="relative z-10 w-full max-w-screen-2xl mx-auto px-4 py-8 md:py-16">
        {@render children()}
    </div>

    <footer class="relative z-10 py-12 border-t border-border/20 backdrop-blur-sm mt-20">
        <div class="max-w-screen-2xl mx-auto px-4 flex flex-col md:flex-row items-center justify-between gap-6 group">
            <div class="flex items-center gap-2">
                <div class="w-8 h-8 rounded-lg bg-muted flex items-center justify-center font-bold text-xs group-hover:bg-primary group-hover:text-primary-foreground transition-all duration-500">
                    AV
                </div>
                <div class="text-xs font-bold tracking-widest text-muted-foreground uppercase">
                    Samuel Moses <span class="text-primary/40 mx-1">/</span> Remote Audio Engine Proxy
                </div>
            </div>
            <div class="text-xxs font-medium text-muted-foreground/60 uppercase tracking-widest">
                Built with Svelte Runes & PortAudio © 2026
            </div>
        </div>
    </footer>
</main>

<style>
    /* Prevent layout shifts */
    :global(html) {
        font-feature-settings: 'cv02', 'cv05', 'cv11', 'ss01';
    }
</style>
