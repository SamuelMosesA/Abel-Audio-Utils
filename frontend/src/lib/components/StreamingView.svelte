<script lang="ts">
    import { audioState } from "../audioState.svelte";
    import { audioVisuals } from "../audioVisuals.svelte";
    import { goto } from "$app/navigation";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import { ChevronLeft, Volume2, Waves, Globe } from "lucide-svelte";
    import { onMount } from "svelte";

    let selectedLang = $state("default");
    let audioSource = $derived(`/api/stream/${selectedLang}`);

    const languages = [
        { id: "default", name: "Original (English)" },
        { id: "tamil", name: "Tamil" },
        { id: "hindi", name: "Hindi" },
        { id: "malayalam", name: "Malayalam" },
        { id: "german", name: "German" },
        { id: "spanish", name: "Spanish" },
        { id: "french", name: "French" },
        { id: "russian", name: "Russian" },
        { id: "chinese", name: "Chinese" },
        { id: "dutch", name: "Dutch" },
        { id: "portugese", name: "Portugese"},
        { id: "korean", name: "Korean"},
        { id: "hungarian", name: "Hungarian"}
    ];

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
            <div class="flex flex-col items-center gap-8">
                <!-- Translation Selector -->
                <div class="w-full max-w-sm space-y-3">
                    <label for="lang-select" class="text-xs font-bold text-muted-foreground uppercase tracking-widest flex items-center gap-2">
                        <Globe class="w-3 h-3" />
                        AI Translation
                    </label>
                    <select 
                        id="lang-select"
                        bind:value={selectedLang}
                        class="w-full bg-muted/50 border border-border rounded-lg px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all appearance-none cursor-pointer"
                    >
                        {#each languages as lang}
                            <option value={lang.id}>{lang.name}</option>
                        {/each}
                    </select>
                    {#if selectedLang !== 'default'}
                        <p class="text-[10px] text-primary/80 animate-pulse font-medium">
                            ✨ Real-time AI translation active
                        </p>
                    {/if}
                </div>

                <!-- Background Audio Stream -->
                <div class="flex flex-col items-center gap-6 w-full">
                    {#key audioSource}
                        <audio 
                            controls 
                            autoplay
                            src={audioSource} 
                            class="w-full max-w-md h-12 opacity-90 hover:opacity-100 transition-opacity"
                        ></audio>
                    {/key}
                    
                    <p class="text-xs text-muted-foreground flex items-center gap-2">
                        <Waves class="w-3 h-3" />
                        Works in background and on lock screen
                    </p>
                </div>
            </div>
        </Card.Content>
    </Card.Root>
</div>
