<script lang="ts">
    import { FileAudio, Cloud, X, Info } from "lucide-svelte";
    import SimpleCard from "./ui/SimpleCard.svelte";
    import SimpleButton from "./ui/SimpleButton.svelte";
    import SimpleInput from "./ui/SimpleInput.svelte";
    import { fileState, type RecordedFile } from "../fileState.svelte";
    import { audioState } from "../audioState.svelte";

    let pushingFile = $state<RecordedFile | null>(null);
    let targetFilename = $state("");

    const openPushDialog = (file: RecordedFile) => {
        pushingFile = file;
        const date = new Date().toISOString().split('T')[0];
        targetFilename = `${date}.wav`;
    };

    const handlePush = async () => {
        if (!pushingFile || !targetFilename) return;
        const res = await fileState.pushToCloud(pushingFile.name, targetFilename);
        if (res.success) {
            pushingFile = null;
        } else {
            alert("Failed to push: " + res.error);
        }
    };

    const formatSize = (bytes: number) => {
        const mb = bytes / (1024 * 1024);
        return `${mb.toFixed(1)} MB`;
    };

    const formatDate = (dateStr: string) => {
        return new Date(dateStr).toLocaleString();
    };
</script>

<SimpleCard class="space-y-8 md:space-y-10">
    <div class="flex items-center gap-3 text-muted-foreground">
        <FileAudio class="w-4 h-4 text-primary" />
        <span class="text-xs font-black uppercase tracking-widest">Master Recording Library</span>
    </div>

    <!-- System Paths -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-6 pb-6 md:pb-10 border-b border-border/40">
        <SimpleCard class="p-4 md:p-5 bg-card/50 space-y-1.5 md:space-y-2 group hover:border-primary/20 transition-colors min-w-0 border-dashed">
            <span class="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground group-hover:text-primary transition-colors">Storage Cluster</span>
            <div class="flex items-center gap-2 md:gap-3 text-xs font-mono text-white opacity-80" title={audioState.storageLocation}>
                <div class="w-1 md:w-1.5 h-1 md:h-1.5 rounded-full bg-primary/40 shrink-0"></div>
                <div class="truncate">{audioState.storageLocation}</div>
            </div>
        </SimpleCard>
        <SimpleCard class="p-4 md:p-5 bg-card/50 space-y-1.5 md:space-y-2 group hover:border-primary/20 transition-colors min-w-0 border-dashed">
            <span class="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground group-hover:text-primary transition-colors">Remote Archive</span>
            <div class="flex items-center gap-2 md:gap-3 text-xs font-mono text-white opacity-80" title={audioState.cloudDriveLocation}>
                <div class="w-1 md:w-1.5 h-1 md:h-1.5 rounded-full bg-primary/40 shrink-0"></div>
                <div class="truncate">{audioState.cloudDriveLocation}</div>
            </div>
        </SimpleCard>
    </div>

    <div class="space-y-4">
        {#if fileState.recordedFiles.length === 0}
            <div class="py-24 text-center border-2 border-dashed border-border/40 rounded-3xl">
                <p class="text-muted-foreground font-medium tracking-wide">No system archives found.</p>
            </div>
        {:else}
            {#each fileState.recordedFiles as file}
                <SimpleCard class="flex flex-col xl:flex-row xl:items-center justify-between hover:border-primary/40 hover:bg-muted/10 transition-all duration-300 gap-6 md:gap-8 active:scale-[0.995] min-w-0 p-4 md:p-6">
                    <div class="flex items-center gap-4 md:gap-6 min-w-0 flex-1">
                        <div class="p-3 md:p-4 bg-muted/30 rounded-xl md:rounded-2xl group-hover:bg-primary/10 transition-colors shrink-0">
                            <FileAudio class="w-6 h-6 md:w-8 md:h-8 text-muted-foreground group-hover:text-primary transition-colors" />
                        </div>
                        <div class="space-y-1 min-w-0">
                            <h3 class="font-bold text-base md:text-lg text-white truncate">{file.name}</h3>
                            <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs font-black uppercase tracking-widest text-muted-foreground/60">
                                <span class="text-primary/80 shrink-0">{formatSize(file.size)}</span>
                                <span class="opacity-20 hidden md:inline">•</span>
                                <span class="shrink-0">{formatDate(file.modTime)}</span>
                            </div>
                        </div>
                    </div>

                    <div class="flex flex-col sm:flex-row items-center gap-4 md:gap-6 w-full xl:w-auto">
                        <audio 
                            controls 
                            src="/api/recordings/{file.name}" 
                            class="h-10 w-full sm:w-[300px] xl:w-[400px] brightness-90 contrast-125"
                        ></audio>
                        <SimpleButton 
                            class="w-full sm:w-auto"
                            onclick={() => openPushDialog(file)}
                        >
                            <Cloud class="w-3.5 h-3.5 md:w-4 md:h-4" />
                            Cloud Push
                        </SimpleButton>
                    </div>
                </SimpleCard>
            {/each}
        {/if}
    </div>
</SimpleCard>

<!-- Push Dialog Modal -->
{#if pushingFile}
    <div class="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/80 backdrop-blur-md animate-in fade-in duration-300">
        <SimpleCard class="w-full max-w-lg bg-[#0a0a0a] border-primary/20 p-10 space-y-10 animate-in zoom-in-[0.98]">
            <div class="flex items-center justify-between">
                <div class="space-y-1">
                    <h2 class="text-2xl font-bold text-white flex items-center gap-3">
                        <Cloud class="w-6 h-6 text-primary" />
                        Cloud Backup
                    </h2>
                    <p class="text-xs font-black uppercase tracking-widest text-muted-foreground">Transferring session to remote storage</p>
                </div>
                <SimpleButton variant="ghost" class="p-2 rounded-full h-auto" onclick={() => pushingFile = null}>
                    <X class="w-6 h-6" />
                </SimpleButton>
            </div>

            <div class="space-y-8">
                <div class="p-6 bg-primary/5 border border-primary/10 rounded-2xl flex gap-4">
                    <Info class="w-6 h-6 text-primary shrink-0" />
                    <p class="text-sm text-muted-foreground leading-relaxed">
                        Pushing to cloud backed-up folder <span class="text-white font-bold">{pushingFile.name}</span>.
                    </p>
                </div>

                <div class="space-y-4">
                    <span class="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground ml-1">Cloud Destination Filename</span>
                    <SimpleInput 
                        bind:value={targetFilename}
                        class="w-full"
                    />
                </div>

                <div class="grid grid-cols-2 gap-4 pt-4">
                    <SimpleButton 
                        variant="secondary"
                        onclick={() => pushingFile = null}
                    >
                        Abandon
                    </SimpleButton>
                    <SimpleButton 
                        onclick={handlePush}
                    >
                        Execute Upload
                    </SimpleButton>
                </div>
            </div>
        </SimpleCard>
    </div>
{/if}

<style>
    audio {
        color-scheme: dark;
    }
</style>
