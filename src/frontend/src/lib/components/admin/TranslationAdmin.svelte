<script lang="ts">
    import { onMount } from "svelte";
    import { getAppContext } from "$lib/audioState.svelte";
    const { ai } = getAppContext();
    import Card from "../ui/Card.svelte";
    import Button from "../ui/Button.svelte";
    import { Languages, Users, RotateCcw, Ban, CheckCircle2 } from "lucide-svelte";

    let refreshing = $state(false);

    onMount(() => {
        void ai.fetchConfig();
        void ai.sync();
    });

    async function handleRefresh() {
        if (refreshing) return;
        refreshing = true;
        try {
            await ai.refreshAIStreams();
        } finally {
            refreshing = false;
        }
    }

    async function toggleMaster() {
        await ai.setAIMaster(!ai.aiMasterEnabled);
    }

    async function toggleKillswitch(langCode: string, currentlyBlocked: boolean) {
        await ai.toggleLanguageKillswitch(langCode, !currentlyBlocked);
    }

    let displayLanguages = $derived.by(() => {
        if (ai.languages && ai.languages.length > 0) {
            return ai.languages;
        }
        return ai.aiConfig.languages.map(l => ({
            code: l.code,
            name: l.name,
            blocked: false,
            active: false,
            listeners: 0
        }));
    });
</script>

<Card 
    title="AI Translation Controls" 
    description="Manage live AI translation pipelines, per-language killswitches, and listener feeds."
>
    <div class="space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3 p-3.5 rounded-lg border border-border bg-muted/20">
            <div class="flex items-center gap-3">
                <div class="h-2.5 w-2.5 rounded-full {ai.aiMasterEnabled ? 'bg-primary' : 'bg-muted-foreground'}"></div>
                <div>
                    <p class="text-xs font-medium text-muted-foreground">Master Control</p>
                    <p class="text-sm font-semibold text-foreground">{ai.aiMasterEnabled ? 'AI Translation Enabled' : 'AI Translation Disabled'}</p>
                </div>
            </div>
            <div class="flex items-center gap-2">
                <Button 
                    variant="outline"
                    size="sm"
                    onclick={handleRefresh}
                    disabled={refreshing}
                    title="Refresh AI streams and listener metrics"
                >
                    <RotateCcw class="w-3.5 h-3.5 mr-1.5 {refreshing ? 'animate-spin' : ''}" />
                    {refreshing ? "Refreshing..." : "Refresh"}
                </Button>
                <Button 
                    variant={ai.aiMasterEnabled ? "destructive" : "primary"}
                    size="sm"
                    onclick={toggleMaster}
                >
                    {ai.aiMasterEnabled ? "Disable AI" : "Enable AI"}
                </Button>
            </div>
        </div>

        {#if displayLanguages.length === 0}
            <div class="py-8 flex flex-col items-center justify-center text-muted-foreground space-y-2 border border-dashed border-border/60 rounded-lg">
                <Languages class="w-6 h-6 opacity-40" />
                <p class="text-xs font-medium">No configured translation languages</p>
            </div>
        {:else}
            <div class="space-y-2">
                <div class="flex items-center justify-between px-1 text-xs font-medium text-muted-foreground">
                    <span>Configured Languages & Feeds</span>
                    <span>Killswitch</span>
                </div>
                {#each displayLanguages as lang (lang.code)}
                    <div class="flex items-center justify-between p-3 border rounded-lg transition-colors {lang.blocked ? 'bg-destructive/5 border-destructive/30' : 'bg-muted/15 border-border'}">
                        <div class="flex items-center gap-3">
                            <div class="w-8 h-8 rounded-md flex items-center justify-center font-bold uppercase text-xs {lang.blocked ? 'bg-destructive/15 text-destructive border border-destructive/20' : 'bg-primary/10 text-primary border border-primary/20'}">
                                {lang.code.substring(0, 2)}
                            </div>
                            <div>
                                <div class="flex items-center gap-2">
                                    <p class="font-medium text-sm text-foreground capitalize">{lang.name}</p>
                                    {#if lang.blocked}
                                        <span class="inline-flex items-center gap-1 text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded bg-destructive/15 text-destructive border border-destructive/20">
                                            <Ban class="w-2.5 h-2.5" /> Blocked
                                        </span>
                                    {:else if lang.active}
                                        <span class="inline-flex items-center gap-1 text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded bg-primary/15 text-primary border border-primary/20">
                                            <CheckCircle2 class="w-2.5 h-2.5" /> Active
                                        </span>
                                    {/if}
                                </div>
                                <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
                                    <Users class="w-3 h-3" />
                                    <span>{lang.listeners} {lang.listeners === 1 ? 'listener' : 'listeners'}</span>
                                </div>
                            </div>
                        </div>

                        <Button
                            variant={lang.blocked ? "outline" : "destructive"}
                            size="sm"
                            class="text-xs"
                            onclick={() => toggleKillswitch(lang.code, lang.blocked)}
                            title={lang.blocked ? "Unblock language translation" : "Killswitch: block language translation"}
                        >
                            {#if lang.blocked}
                                Unblock
                            {:else}
                                <Ban class="w-3.5 h-3.5 mr-1" /> Killswitch
                            {/if}
                        </Button>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</Card>
