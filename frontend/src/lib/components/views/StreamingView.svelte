<script lang="ts">
    import { getAppContext } from "../../audioState.svelte";
    const { ai, ui } = getAppContext();
    import { goto } from "$app/navigation";
    import { ChevronLeft, Volume2, Waves, Globe, MessageSquare } from "lucide-svelte";
    import { onMount, tick } from "svelte";
    import { fly, fade } from "svelte/transition";
    import Button from "../ui/Button.svelte";
    import Card from "../ui/Card.svelte";
    import LanguageSelector from "../ai/LanguageSelector.svelte";

    let { lang = "default" } = $props();
    let subtitleState = $state({ 
        tokenList: [] as {id: string, text: string}[],
        totalTokens: 0 
    });
    
    // Smooth delivery queue
    let tokenQueue: string[] = [];
    let isProcessingQueue = false;

    async function processQueue() {
        if (isProcessingQueue || tokenQueue.length === 0) return;
        isProcessingQueue = true;
        
        while (tokenQueue.length > 0) {
            const text = tokenQueue.shift()!;
            // Split by space but preserve the spaces in the output
            const segments = text.split(/(\s+)/);
            
            for (const segment of segments) {
                if (!segment) continue;
                
                subtitleState.tokenList = [...subtitleState.tokenList, {
                    id: Math.random().toString(36).substring(2),
                    text: segment
                }]; // Unlimited history
                
                // Pace based on segment type: spaces are faster, words slightly slower
                const delay = segment.trim() === "" ? 20 : 60;
                await new Promise(r => setTimeout(r, delay));
                
                await tick();
                if (scrollContainerRef && autoScrollEnabled) {
                    scrollContainerRef.scrollTop = scrollContainerRef.scrollHeight;
                }
            }
        }
        isProcessingQueue = false;
    }

    let audioSource = $derived(`/api/audio/stream/${lang}`);
    
    let eventSource: EventSource | null = null;
    let scrollContainerRef = $state<HTMLElement | null>(null);
    let autoScrollEnabled = $state(true);

    ui.currentView = "stream";

    $effect(() => {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        
        // Subtitles are now always on if AI is enabled
        if (ai.aiMasterEnabled) {
            eventSource = new EventSource(`/api/ai/subtitles?lang=${lang}`);
            eventSource.onmessage = async (e) => {
                try {
                    const data = JSON.parse(e.data);
                    if (data.tokens > 0) {
                        subtitleState.totalTokens = data.tokens;
                    }
                    if (data.error) {
                         tokenQueue.push(` [Notice: ${data.error}] `);
                    } else if (data.text) {
                        tokenQueue.push(data.text);
                    }
                    processQueue();
                } catch (err) {
                    console.error("Subtitle parse error:", err, e.data);
                }
            };
        } else {
            subtitleState.tokenList = [];
        }

        return () => {
            if (eventSource) eventSource.close();
        };
    });

    function handleScroll(e: Event) {
        if (!scrollContainerRef) return;
        const target = e.target as HTMLElement;
        const isAtBottom = Math.abs(target.scrollHeight - target.clientHeight - target.scrollTop) < 20;
        autoScrollEnabled = isAtBottom;
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
        <Button variant="ghost" onclick={() => goto("/")} size="sm">
            <ChevronLeft class="w-4 h-4 mr-2" /> Back
        </Button>
        <div class="flex items-center gap-2 px-3 py-1 rounded-full bg-primary/10 border border-primary/30 text-primary text-xxs font-bold tracking-widest uppercase">
            <span class="w-1.5 h-1.5 rounded-full bg-primary animate-pulse"></span>
            AI Broadcast
        </div>
    </header>
 
    <Card title="AI Stream View ({ai.resolveLanguageName(lang)})" class="glass shadow-2xl overflow-hidden border-primary/5">
        <div class="space-y-8 p-4">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-8 items-start">
                <!-- Left: Audio Stream -->
                <div class="space-y-8 h-full flex flex-col justify-center">
                    <div class="space-y-4">
                        <span class="text-xxs font-black uppercase tracking-extra text-muted-foreground">Audio Stream</span>
                        {#key audioSource}
                            <audio 
                                controls 
                                autoplay
                                src={audioSource} 
                                class="w-full h-10 rounded-lg opacity-80 hover:opacity-100 transition-opacity"
                            ></audio>
                        {/key}
                        <div class="flex items-center gap-2 text-xxs text-muted-foreground/60 font-medium italic">
                            <Waves class="w-3 h-3" />
                            Live sync stream
                        </div>
                    </div>
                </div>
 
                <!-- Right: Subtitles -->
                <div class="space-y-4">
                    <label class="text-xxs font-black uppercase tracking-extra text-muted-foreground flex justify-between">
                        <div class="flex items-center gap-2">
                             <span>AI Subtitles</span>
                             {#if subtitleState.totalTokens > 0}
                                <span class="px-1.5 py-0.5 rounded bg-primary/10 border border-primary/20 text-primary animate-in fade-in transition-all">
                                    {subtitleState.totalTokens.toLocaleString()} tokens
                                </span>
                             {/if}
                        </div>
                        {#if !autoScrollEnabled}
                            <button onclick={resumeAutoScroll} class="text-primary hover:underline transition-all">Resume Scroll</button>
                        {/if}
                    </label>
                    
                    <div class="bg-black/40 rounded-xl border border-border/50 h-[500px] md:h-[600px] flex flex-col p-4 relative overflow-hidden group">
                        <div 
                            class="flex-1 overflow-y-auto custom-scrollbar pr-2 space-y-2 touch-pan-y"
                            style="-webkit-overflow-scrolling: touch;"
                            bind:this={scrollContainerRef}
                            onscroll={handleScroll}
                        >
                            {#if !ai.aiMasterEnabled}
                                <div class="h-full flex flex-col items-center justify-center text-center p-4">
                                    <Globe class="w-8 h-8 text-muted/20 mb-2" />
                                    <p class="text-xs text-muted-foreground font-medium">Translation unavailable</p>
                                </div>
                            {:else if subtitleState.tokenList.length === 0}
                                <div class="h-full flex items-center justify-center">
                                    <div class="flex gap-1">
                                        {#each Array(3) as _}
                                            <div class="w-1 h-3 bg-primary/20 rounded-full animate-bounce"></div>
                                        {/each}
                                    </div>
                                </div>
                            {:else}
                                <div class="text-left flex flex-wrap content-start items-start justify-start">
                                    {#each subtitleState.tokenList as item (item.id)}
                                        <span 
                                            in:fade={{ duration: 150 }}
                                            class="inline whitespace-pre-wrap text-sm md:text-base font-bold text-white/90 transition-all duration-300 animate-in zoom-in-95"
                                        >{item.text}</span>
                                    {/each}
                                </div>
                            {/if}
                        </div>
                        <div class="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-black/60 to-transparent pointer-events-none"></div>
                    </div>
                </div>
            </div>
 
            <div class="pt-4 border-t border-border/50 text-xxs text-center text-muted-foreground/40 uppercase tracking-ultra font-bold">
                Direct Console Interface • AI Assisted Access
            </div>
        </div>
    </Card>
</div>

<style>
    :global(.custom-scrollbar::-webkit-scrollbar) { width: 0.5rem; height: 0.5rem; }
    :global(.custom-scrollbar::-webkit-scrollbar-thumb) {
        background: rgba(255, 255, 255, 0.3);
        border-radius: 0.5rem;
        border: 2px solid transparent;
        background-clip: content-box;
    }
    :global(.custom-scrollbar::-webkit-scrollbar-track) {
        background: rgba(0, 0, 0, 0.1);
    }
    /* Ensure scrollbar is visible on mobile if it has content */
    @media (max-width: 768px) {
        :global(.custom-scrollbar) {
            scrollbar-width: auto;
            -webkit-overflow-scrolling: touch;
        }
    }
</style>
