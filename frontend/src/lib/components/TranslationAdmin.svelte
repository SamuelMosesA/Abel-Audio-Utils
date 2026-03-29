<script lang="ts">
    import { audioState } from "$lib/audioState.svelte";
    import { audioConfig } from "$lib/audioConfig.svelte";
    import SimpleCard from "./ui/SimpleCard.svelte";
    import SimpleButton from "./ui/SimpleButton.svelte";
    import { Languages, XCircle, Users } from "lucide-svelte";
    import { onMount } from "svelte";

    onMount(() => {
        const interval = setInterval(() => {
            if (audioState.wsConnected) {
                audioState.syncStatus();
            }
        }, 3000);
        return () => clearInterval(interval);
    });

    async function handleStop(lang: string) {
        if (confirm(`Are you sure you want to stop the ${lang} translation?`)) {
            await audioConfig.stopTranslation(lang);
        }
    }

    async function toggleMaster() {
        await audioConfig.setGeminiMaster(!audioState.geminiMasterEnabled);
    }
</script>

<SimpleCard class="space-y-6 md:space-y-8 text-white">
    <div class="flex items-center justify-between border-b border-border/40 pb-4">
        <div class="flex items-center gap-3 text-muted-foreground">
            <Languages class="w-4 h-4 text-primary" />
            <span class="text-xs font-black uppercase tracking-widest">Active Translations</span>
        </div>
        <div class="flex items-center gap-4">
            <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg border transition-all {audioState.geminiMasterEnabled ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400' : 'bg-red-500/10 border-red-500/30 text-red-400'}">
                <span class="w-1.5 h-1.5 rounded-full {audioState.geminiMasterEnabled ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'}"></span>
                <span class="text-[10px] font-black uppercase tracking-wider">Gemini API: {audioState.geminiMasterEnabled ? 'Active' : 'Disabled'}</span>
            </div>
            <SimpleButton 
                variant={audioState.geminiMasterEnabled ? "destructive" : "primary"}
                class="h-8 px-4 text-[10px]"
                onclick={toggleMaster}
            >
                {audioState.geminiMasterEnabled ? 'Disable Gemini' : 'Enable Gemini'}
            </SimpleButton>
        </div>
    </div>

    {#if audioState.translations.length === 0}
        <div class="py-12 flex flex-col items-center justify-center text-muted-foreground space-y-3 opacity-60">
            <div class="p-3 bg-muted/20 rounded-full">
                <Languages class="w-6 h-6" />
            </div>
            <p class="text-sm font-medium italic">No active translation sessions</p>
        </div>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            {#each audioState.translations as session}
                <div class="flex items-center justify-between p-4 bg-black/40 border border-border/60 rounded-xl hover:border-primary/30 transition-all group">
                    <div class="flex items-center gap-4">
                        <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center font-black text-primary uppercase text-sm border border-primary/10">
                            {session.language.substring(0, 2)}
                        </div>
                        <div class="flex flex-col">
                            <span class="font-bold text-sm tracking-tight capitalize">{session.language}</span>
                            <div class="flex items-center gap-1.5 text-[10px] text-muted-foreground font-medium uppercase tracking-wider">
                                <Users class="w-3 h-3" />
                                <span>External Feed</span>
                                {#if session.subtitles}
                                    <span class="mx-1">•</span>
                                    <span class="text-primary font-bold">Subtitles ON</span>
                                {/if}
                            </div>
                        </div>
                    </div>
                    
                    <SimpleButton 
                        variant="destructive" 
                        class="px-3 h-9 opacity-0 group-hover:opacity-100 transition-opacity"
                        onclick={() => handleStop(session.language)}
                    >
                        <XCircle class="w-4 h-4 mr-1.5" />
                        Kill
                    </SimpleButton>
                </div>
            {/each}
        </div>
    {/if}
</SimpleCard>
