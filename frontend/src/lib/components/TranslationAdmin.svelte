<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { ai, audio } = getAppContext();
    import SimpleCard from "./ui/SimpleCard.svelte";
    import SimpleButton from "./ui/SimpleButton.svelte";
    import { Languages, XCircle, Users } from "lucide-svelte";


    async function handleStop(lang: string) {
        if (confirm(`Are you sure you want to stop the ${lang} translation?`)) {
            await ai.stopTranslation(lang);
        }
    }

    async function toggleMaster() {
        await ai.setGeminiMaster(!ai.geminiMasterEnabled);
    }
</script>

<SimpleCard class="space-y-6 md:space-y-8 text-white">
    <div
        class="flex items-center justify-between border-b border-border/40 pb-4"
    >
        <div class="flex items-center gap-3 text-muted-foreground">
            <Languages class="w-4 h-4 text-primary" />
            <span class="text-xs font-black uppercase tracking-widest"
                >Active Translations</span
            >
        </div>
        <div class="flex items-center gap-4">
            <div
                class="flex items-center gap-2 px-3 py-1.5 rounded-lg border {ai.geminiMasterEnabled
                    ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400'
                    : 'bg-red-500/10 border-red-500/30 text-red-400'}"
            >
                <span
                    class="w-1.5 h-1.5 rounded-full {ai.geminiMasterEnabled
                        ? 'bg-emerald-500'
                        : 'bg-red-500'}"
                ></span>
                <span class="text-xs font-black uppercase tracking-widest"
                    >Gemini API: {ai.geminiMasterEnabled
                        ? "Active"
                        : "Disabled"}</span
                >
            </div>
            <SimpleButton
                variant={ai.geminiMasterEnabled
                    ? "destructive"
                    : "primary"}
                class="h-9 px-4 text-xs"
                onclick={toggleMaster}
            >
                {ai.geminiMasterEnabled
                    ? "Disable Gemini"
                    : "Enable Gemini"}
            </SimpleButton>
        </div>
    </div>

    {#if ai.translations.length === 0}
        <div
            class="py-12 flex flex-col items-center justify-center text-muted-foreground space-y-3 opacity-60"
        >
            <div class="p-3 bg-muted/20 rounded-full">
                <Languages class="w-6 h-6" />
            </div>
            <p class="text-sm font-medium italic">
                No active translation sessions
            </p>
        </div>
    {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            {#each ai.translations as session}
                <div
                    class="flex items-center justify-between p-5 bg-muted/30 border border-border/40 rounded-xl group"
                >
                    <div class="flex items-center gap-4">
                        <div
                            class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center font-black text-primary uppercase text-sm border border-primary/10"
                        >
                            {session.language.substring(0, 2)}
                        </div>
                        <div class="flex flex-col">
                            <span
                                class="font-bold text-sm tracking-tight capitalize"
                                >{session.language}</span
                            >
                            <div
                                class="flex items-center gap-2 text-xs text-muted-foreground font-medium uppercase tracking-widest"
                            >
                                <Users class="w-3 h-3" />
                                <span>External Feed</span>
                                {#if session.subtitles}
                                    <span class="mx-1">•</span>
                                    <span class="text-primary font-bold"
                                        >Subtitles ON</span
                                    >
                                {/if}
                            </div>
                        </div>
                    </div>

                    <SimpleButton
                        variant="destructive"
                        class="px-3 h-9"
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
