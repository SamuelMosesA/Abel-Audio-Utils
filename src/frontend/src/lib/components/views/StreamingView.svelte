<script lang="ts">
    import { getAppContext } from "../../audioState.svelte";
    const { ai, ui } = getAppContext();
    import { goto } from "$app/navigation";
    import { ChevronLeft, Volume2, Globe } from "lucide-svelte";
    import { onMount, tick } from "svelte";
    import { fade } from "svelte/transition";
    import Button from "../ui/Button.svelte";
    import Card from "../ui/Card.svelte";
    import LiveAudioPlayer from "../audio/LiveAudioPlayer.svelte";

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
            const segments = text.split(/(\s+)/);
            
            for (const segment of segments) {
                if (!segment) continue;
                
                subtitleState.tokenList = [...subtitleState.tokenList, {
                    id: Math.random().toString(36).substring(2),
                    text: segment
                }];
                
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

    let audioSource = $derived(`/api/audio/hls/${lang}/index.m3u8`);
    
    let eventSource: EventSource | null = null;
    let scrollContainerRef = $state<HTMLElement | null>(null);
    let autoScrollEnabled = $state(true);

    ui.currentView = "stream";

    $effect(() => {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        
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

<div class="space-y-6">
    <header class="flex items-center justify-between pb-4 border-b border-border/40">
        <Button variant="ghost" onclick={() => goto("/")} size="sm">
            <ChevronLeft class="w-4 h-4 mr-1" /> Back to Broadcast
        </Button>
        <div class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-primary/10 border border-primary/20 text-primary text-xs font-medium">
            <span class="w-1.5 h-1.5 rounded-full bg-primary animate-pulse"></span>
            <span>Live Stream</span>
        </div>
    </header>
 
    <Card 
        title={`Live Stream: ${ai.resolveLanguageName(lang)}`}
        description="Listen to translated speech and follow live subtitles."
    >
        <div class="grid grid-cols-1 md:grid-cols-2 gap-8 items-start">
            <!-- Audio Section -->
            <div class="space-y-4">
                <div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                    <Volume2 class="w-4 h-4 text-primary" />
                    <span>Synchronized Audio</span>
                </div>
                {#key audioSource}
                    <LiveAudioPlayer
                        src={audioSource}
                        label={`Live ${ai.resolveLanguageName(lang)} audio`}
                    />
                {/key}

            </div>

            <!-- Subtitles Section -->
            <div class="space-y-3">
                <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                    <div class="flex items-center gap-2">
                        <span>Live AI Subtitles</span>
                        {#if subtitleState.totalTokens > 0}
                            <span class="px-1.5 py-0.5 rounded text-xs font-mono bg-secondary text-secondary-foreground border border-border">
                                {subtitleState.totalTokens.toLocaleString()} tokens
                            </span>
                        {/if}
                    </div>
                    {#if !autoScrollEnabled}
                        <button onclick={resumeAutoScroll} class="text-xs text-primary hover:underline">
                            Auto-scroll paused (click to resume)
                        </button>
                    {/if}
                </div>
                
                <div class="bg-background rounded-lg border border-border h-[420px] md:h-[500px] flex flex-col p-4 relative overflow-hidden">
                    <div 
                        class="flex-1 overflow-y-auto pr-2 space-y-2"
                        bind:this={scrollContainerRef}
                        onscroll={handleScroll}
                    >
                        {#if !ai.aiMasterEnabled}
                            <div class="h-full flex flex-col items-center justify-center text-center p-4">
                                <Globe class="w-8 h-8 text-muted-foreground/30 mb-2" />
                                <p class="text-xs text-muted-foreground">AI translation is currently disabled.</p>
                            </div>
                        {:else if subtitleState.tokenList.length === 0}
                            <div class="h-full flex items-center justify-center text-xs text-muted-foreground">
                                <span>Listening for speech…</span>
                            </div>
                        {:else}
                            <div class="text-left flex flex-wrap content-start items-start justify-start">
                                {#each subtitleState.tokenList as item (item.id)}
                                    <span 
                                        in:fade={{ duration: 150 }}
                                        class="inline whitespace-pre-wrap text-sm leading-relaxed text-foreground"
                                    >{item.text}</span>
                                {/each}
                            </div>
                        {/if}
                    </div>
                </div>
            </div>
        </div>
    </Card>
</div>
