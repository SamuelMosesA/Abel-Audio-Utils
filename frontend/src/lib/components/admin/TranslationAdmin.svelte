<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { ai } = getAppContext();
    import Card from "../ui/Card.svelte";
    import Button from "../ui/Button.svelte";
    import { Languages, XCircle, Users, Activity } from "lucide-svelte";

    async function handleStop(langCode: string) {
        const langName = ai.resolveLanguageName(langCode);
        if (confirm(`Are you sure you want to stop the ${langName} translation?`)) {
            await ai.stopTranslation(langCode);
        }
    }

    async function toggleMaster() {
        await ai.setAIMaster(!ai.aiMasterEnabled);
    }
</script>

<Card title="AI Translation Control">
    <div class="space-y-6">
        <div class="flex items-center justify-between bg-muted/30 p-4 rounded-xl border border-border/40">
            <div class="flex items-center gap-3">
                <div class="flex h-3 w-3 rounded-full {ai.aiMasterEnabled ? 'bg-primary animate-pulse' : 'bg-destructive'}"></div>
                <div class="flex flex-col">
                    <span class="text-xxs font-black uppercase tracking-widest text-muted-foreground">AI Master Switch</span>
                    <span class="text-sm font-bold text-white">{ai.aiMasterEnabled ? 'SYSTEM ACTIVE' : 'SYSTEM DISABLED'}</span>
                </div>
            </div>
            <Button 
                variant={ai.aiMasterEnabled ? "destructive" : "primary"}
                size="sm"
                onclick={toggleMaster}
            >
                {ai.aiMasterEnabled ? "Disable" : "Enable"} AI
            </Button>
        </div>

        {#if ai.translations.length === 0}
            <div class="py-10 flex flex-col items-center justify-center text-muted-foreground/40 space-y-2">
                <Languages class="w-8 h-8 opacity-20" />
                <p class="text-xs font-bold uppercase tracking-widest">No Active Sessions</p>
            </div>
        {:else}
            <div class="grid grid-cols-1 gap-3">
                {#each ai.translations as session}
                    <div class="flex items-center justify-between p-4 bg-muted/20 border border-border/40 rounded-xl hover:border-primary/20 transition-colors group">
                        <div class="flex items-center gap-4">
                            <div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center font-black text-primary uppercase text-sm">
                                {session.language.substring(0, 2)}
                            </div>
                            <div class="flex flex-col">
                                <span class="font-bold text-sm tracking-tight capitalize text-white">{ai.resolveLanguageName(session.language)}</span>
                                <div class="flex items-center gap-2 text-xxs text-muted-foreground font-black uppercase tracking-widest">
                                    <Users class="w-3 h-3" />
                                    <span>External Feed</span>
                                    <span class="mx-1">•</span>
                                    <span class="text-primary">Subtitles Forced</span>
                                </div>
                            </div>
                        </div>

                        <Button
                            variant="destructive"
                            size="icon"
                            onclick={() => handleStop(session.language)}
                            title="Kill Session"
                        >
                            <XCircle class="w-4 h-4" />
                        </Button>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</Card>
