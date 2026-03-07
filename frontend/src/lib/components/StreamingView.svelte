<script lang="ts">
    import { audioState } from "../audioState.svelte";
    import { audioVisuals } from "../audioVisuals.svelte";
    import { goto } from "$app/navigation";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import { ChevronLeft, Volume2, Waves } from "lucide-svelte";
    import { onMount } from "svelte";

    onMount(() => {
        audioState.currentView = "stream";
    });
</script>

<div class="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
    <header class="flex items-center justify-between">
        <Button variant="ghost" onclick={() => goto("/")} class="gap-2">
            <ChevronLeft class="w-4 h-4" />
            Back to Home
        </Button>
        <div class="flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-500 text-xs font-bold">
            <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
            LIVE STREAM
        </div>
    </header>

    <Card.Root class="bg-card/40 border-border backdrop-blur-xl shadow-2xl overflow-hidden">
        <Card.Header class="text-center space-y-4">
            <div class="mx-auto p-4 bg-primary/20 rounded-full w-fit shadow-[0_0_30px_hsl(var(--primary)/0.2)]">
                <Volume2 class="w-12 h-12 text-primary" />
            </div>
            <div>
                <Card.Title class="text-2xl font-bold">Live Audio Stream</Card.Title>
                <Card.Description>Listen to the direct feed from the recorder</Card.Description>
            </div>
        </Card.Header>
        <Card.Content class="space-y-8 pb-12">
            <!-- Background Audio Stream -->
            <div class="flex flex-col items-center gap-6">
                <audio 
                    controls 
                    autoplay
                    src="/api/stream" 
                    class="w-full max-w-md h-12 opacity-90 hover:opacity-100 transition-opacity"
                ></audio>
                
                <p class="text-xs text-muted-foreground flex items-center gap-2">
                    <Waves class="w-3 h-3" />
                    Works in background and on lock screen
                </p>
            </div>
        </Card.Content>
    </Card.Root>
</div>
