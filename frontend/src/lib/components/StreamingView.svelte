<script lang="ts">
    import { getAppContext } from "../audioState.svelte";
    const { ai, ui, audio } = getAppContext();
    import { goto } from "$app/navigation";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import { ChevronLeft, Volume2, Waves, Globe } from "lucide-svelte";
    import { onMount, tick } from "svelte";
    import { fade, fly } from "svelte/transition";

    let selectedLang = $state("default");
    let showSubtitles = $state(false);
    let subtitleState = $state({ tokenList: [] as {id: string, text: string}[] });
    let audioSource = $derived(`/api/audio/stream/${selectedLang}`);
    
    let eventSource: EventSource | null = null;
    let scrollContainerRef = $state<HTMLElement | null>(null);
    let autoScrollEnabled = $state(true);

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

    ui.currentView = "stream";

    $effect(() => {
        return () => {
            if (eventSource) eventSource.close();
        };
    });

    $effect(() => {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        
        if (showSubtitles && ai.geminiMasterEnabled) {
            eventSource = new EventSource(`/api/ai/subtitles?lang=${selectedLang}`);
            eventSource.onmessage = async (e) => {
                try {
                    const data = JSON.parse(e.data);
                    if (data.error) {
                        subtitleState.tokenList = [...subtitleState.tokenList, { id: Math.random().toString(), text: `[Notice] ${data.error}` }];
                    } else if (data.text) {
                        const newTokens = [{
                            id: Math.random().toString(36).substring(2), 
                            text: data.text 
                        }];
                        
                        let updatedTokens = [...subtitleState.tokenList, ...newTokens];
                        subtitleState.tokenList = updatedTokens;
                        console.log(subtitleState.tokenList);
                        
                        await tick();
                        if (scrollContainerRef && autoScrollEnabled) {
                            scrollContainerRef.scrollTop = scrollContainerRef.scrollHeight;
                        }
                    }
                } catch (err) {
                    console.error("Subtitle parse error:", err, e.data);
                }
            };
            eventSource.onerror = () => {
                subtitleState.tokenList = [...subtitleState.tokenList, { id: "error", text: "Connection lost. Reconnecting..." }];
            };
        } else {
            subtitleState.tokenList = [];
        }
    });

    function toggleSubtitles() {
        showSubtitles = !showSubtitles;
    }

    function handleScroll(e: Event) {
        if (!scrollContainerRef) return;
        
        const target = e.target as HTMLElement;
        const isAtBottom = Math.abs(target.scrollHeight - target.clientHeight - target.scrollTop) < 10;
        
        if (!isAtBottom) {
            autoScrollEnabled = false;
        } else {
            autoScrollEnabled = true;
        }
    }
    
    function resumeAutoScroll() {
        autoScrollEnabled = true;
        if (scrollContainerRef) {
            scrollContainerRef.scrollTop = scrollContainerRef.scrollHeight;
        }
    }
</script>

<div class="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
    <header class="flex items-center justify-between">
        <Button variant="ghost" onclick={() => goto("/")} class="gap-2">
            <ChevronLeft class="w-4 h-4" />
            Back to Home
        </Button>
        <div class="flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-500 text-xs font-bold">
            <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
            LIVE STREAM
        </div>
    </header>

    <Card.Root class="bg-card/40 border-border backdrop-blur-xl shadow-lg overflow-hidden">
        <Card.Header class="text-center space-y-4">
            <div class="mx-auto p-4 bg-primary/20 rounded-full w-fit">
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
                        <div class="flex flex-col gap-2">
                            <p class="text-[10px] text-primary/80 font-medium">
                                {#if selectedLang !== 'default'}
                                    ✨ Real-time AI translation active
                                {:else}
                                    ✨ Live transcription active
                                {/if}
                            </p>
                            
                            <button 
                                class="flex items-center gap-2 group w-fit"
                                onclick={toggleSubtitles}
                            >
                                <div class="w-8 h-4 rounded-full border border-border/60 p-0.5 transition-colors {showSubtitles ? 'bg-primary border-primary' : 'bg-muted'} relative">
                                    <div class="absolute top-0.5 bottom-0.5 w-3 rounded-full bg-white transition-all {showSubtitles ? 'right-0.5' : 'left-0.5'} shadow-sm"></div>
                                </div>
                                <span class="text-xs font-bold uppercase tracking-widest {showSubtitles ? 'text-primary' : 'text-muted-foreground'}">Subtitles</span>
                                </button>
                        </div>
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

                <!-- Subtitles Area -->
                {#if showSubtitles}
                    <div class="w-full max-w-lg mt-8 animate-in fade-in slide-in-from-bottom-2 duration-300">
                        <div class="bg-black/60 border border-border/40 backdrop-blur-md rounded-2xl p-6 shadow-xl relative overflow-hidden group">
                            <!-- Status bar -->
                            <div class="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-primary/20 via-primary to-primary/20 opacity-40 group-hover:opacity-100 transition-opacity"></div>
                            
                            <div class="flex items-center justify-between mb-4">
                                <div class="flex items-center gap-2 text-xs font-black uppercase tracking-widest text-primary/60">
                                    <span class="w-1.5 h-1.5 rounded-full bg-primary"></span>
                                    Live Subtitles ({languages.find(l => l.id === selectedLang)?.name || selectedLang})
                                </div>
                                <div class="text-xs text-muted-foreground font-medium italic">
                                    {subtitleState.tokenList.length} tokens
                                </div>
                            </div>
                            <div 
                                class="h-80 custom-scrollbar overflow-y-auto relative" 
                                id="subtitle-container" 
                                bind:this={scrollContainerRef}
                                onscroll={handleScroll}
                            >
                                {#if !ai.geminiMasterEnabled}
                                    <div class="h-full flex items-center justify-center">
                                        <p class="text-sm text-destructive font-medium italic">
                                            Live translation is currently disabled by administrator.
                                        </p>
                                    </div>
                                {:else if subtitleState.tokenList.length === 0}
                                    <div class="h-full flex items-center justify-center">
                                        <div class="flex items-center gap-3">
                                            <div class="flex gap-1">
                                                <div class="w-1.5 h-1.5 rounded-full bg-primary/40"></div>
                                                <div class="w-1.5 h-1.5 rounded-full bg-primary/40"></div>
                                                <div class="w-1.5 h-1.5 rounded-full bg-primary/40"></div>
                                            </div>
                                            <p class="text-[10px] text-muted-foreground/60 font-medium uppercase tracking-widest mt-0.5">
                                                Listening...
                                            </p>
                                        </div>
                                    </div>
                                {:else}
                                    <div class="w-full h-full text-left">
                                        {#each subtitleState.tokenList as item (item.id)}
                                            <span 
                                                in:fly={{ y: 5, duration: 200 }}
                                                class="inline whitespace-pre-wrap text-base md:text-xl font-black tracking-tight leading-relaxed text-white drop-shadow-lg"
                                            >{item.text}</span>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                            
                            <!-- Resume Auto-scroll Button -->
                            {#if !autoScrollEnabled}
                                <div class="absolute bottom-6 left-1/2 -translate-x-1/2 z-10" in:fade={{ duration: 150 }} out:fade={{ duration: 150 }}>
                                    <button 
                                        class="bg-primary/90 hover:bg-primary text-black text-xs font-bold px-4 py-1.5 rounded-full shadow-lg backdrop-blur-md transition-all flex items-center gap-2"
                                        onclick={resumeAutoScroll}
                                    >
                                        <span class="w-2 h-2 rounded-full bg-black"></span>
                                        Resume scroll
                                    </button>
                                </div>
                            {/if}
                            
                            <!-- Bottom Gradient for focus -->
                            <div class="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-black/80 to-transparent pointer-events-none"></div>
                        </div>
                    </div>
                {/if}
            </div>
        </Card.Content>
    </Card.Root>
</div>

<style>
    :global(.custom-scrollbar::-webkit-scrollbar) {
        width: 4px;
    }
    :global(.custom-scrollbar::-webkit-scrollbar-track) {
        background: transparent;
    }
    :global(.custom-scrollbar::-webkit-scrollbar-thumb) {
        background: rgba(255, 255, 255, 0.1);
        border-radius: 10px;
    }
</style>
