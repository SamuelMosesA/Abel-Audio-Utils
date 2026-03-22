<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { ui } = getAppContext();
    import { fade, fly } from "svelte/transition";
    import { Bell, X } from "lucide-svelte";
</script>

{#if ui.notification}
    <div 
        class="fixed top-6 left-1/2 -translate-x-1/2 z-[100] w-[calc(100%-2rem)] max-w-md"
        in:fly={{ y: -20, duration: 400 }}
        out:fade={{ duration: 200 }}
    >
        <div class="bg-card/80 backdrop-blur-xl border border-primary/20 rounded-2xl p-4 flex items-center gap-4">
            <div class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                <Bell class="w-5 h-5 text-primary" />
            </div>
            <div class="flex-grow">
                <p class="text-xs font-bold uppercase tracking-widest text-primary/60 mb-1">Update: {ui.notification.section}</p>
                <p class="text-sm font-medium text-foreground">{ui.notification.message}</p>
            </div>
            <button 
                onclick={() => ui.notification = null}
                class="w-8 h-8 rounded-full hover:bg-muted flex items-center justify-center transition-colors"
                aria-label="Close notification"
            >
                <X class="w-4 h-4 text-muted-foreground" />
            </button>
        </div>
    </div>
{/if}
