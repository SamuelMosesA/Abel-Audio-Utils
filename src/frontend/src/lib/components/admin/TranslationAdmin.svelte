<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { ai } = getAppContext();
    import Card from "../ui/Card.svelte";
    import Button from "../ui/Button.svelte";
    import { Languages, XCircle, Users } from "lucide-svelte";

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

<Card 
    title="AI Translation Sessions" 
    description="Manage live AI translation pipelines and subtitle streams."
>
    <div class="space-y-4">
        <div class="flex items-center justify-between p-3.5 rounded-lg border border-border bg-muted/20">
            <div class="flex items-center gap-3">
                <div class="h-2.5 w-2.5 rounded-full {ai.aiMasterEnabled ? 'bg-primary' : 'bg-muted-foreground'}"></div>
                <div>
                    <p class="text-xs font-medium text-muted-foreground">Master Control</p>
                    <p class="text-sm font-semibold text-foreground">{ai.aiMasterEnabled ? 'AI Translation Enabled' : 'AI Translation Disabled'}</p>
                </div>
            </div>
            <Button 
                variant={ai.aiMasterEnabled ? "destructive" : "primary"}
                size="sm"
                onclick={toggleMaster}
            >
                {ai.aiMasterEnabled ? "Disable AI" : "Enable AI"}
            </Button>
        </div>

        {#if ai.translations.length === 0}
            <div class="py-8 flex flex-col items-center justify-center text-muted-foreground space-y-2 border border-dashed border-border/60 rounded-lg">
                <Languages class="w-6 h-6 opacity-40" />
                <p class="text-xs font-medium">No active translation sessions</p>
            </div>
        {:else}
            <div class="space-y-2">
                {#each ai.translations as session}
                    <div class="flex items-center justify-between p-3 bg-muted/15 border border-border rounded-lg">
                        <div class="flex items-center gap-3">
                            <div class="w-8 h-8 rounded-md bg-primary/10 border border-primary/20 flex items-center justify-center font-bold text-primary uppercase text-xs">
                                {session.language.substring(0, 2)}
                            </div>
                            <div>
                                <p class="font-medium text-sm text-foreground capitalize">{ai.resolveLanguageName(session.language)}</p>
                                <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
                                    <Users class="w-3 h-3" />
                                    <span>Active Listener Feed</span>
                                </div>
                            </div>
                        </div>

                        <Button
                            variant="ghost"
                            size="icon"
                            class="text-muted-foreground hover:text-destructive"
                            onclick={() => handleStop(session.language)}
                            title="Stop Translation Session"
                        >
                            <XCircle class="w-4 h-4" />
                        </Button>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</Card>
