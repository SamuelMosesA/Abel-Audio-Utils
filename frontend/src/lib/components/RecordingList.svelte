<script lang="ts">
    import { fileState, type RecordedFile } from "../fileState.svelte";
    import { audioState } from "../audioState.svelte";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import { Input } from "$lib/components/ui/input/index.js";
    import { Label } from "$lib/components/ui/label/index.js";
    import { Download, Cloud, X, Info } from "lucide-svelte";
    import { cn } from "$lib/utils/utils";

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
            // Optionally show success toast
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

<Card.Root class="bg-card/40 border-border backdrop-blur-xl shadow-2xl">
    <Card.Header>
        <Card.Title class="flex items-center gap-2 text-card-foreground/80 text-lg">
            <Download class="w-5 h-5 text-primary" />
            Recorded Files
        </Card.Title>
        <Card.Description class="text-muted-foreground text-s">
            Manage and playback your recordings. Push to cloud drive for sharing.
        </Card.Description>
    </Card.Header>
    <Card.Content class="space-y-6">
        <!-- System Paths (Integrated) -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3 pb-2">
            <div class="flex items-center gap-3 p-3 bg-muted/20 border border-border/50 rounded-xl text-[10px] font-mono group/path overflow-hidden">
                <span class="px-2 py-0.5 rounded bg-primary/10 text-primary border border-primary/20 font-black shrink-0 uppercase">Storage</span>
                <span class="text-muted-foreground/80 truncate flex-1 hover:text-primary transition-colors cursor-default" title={audioState.storageLocation}>
                    {audioState.storageLocation}
                </span>
            </div>
            <div class="flex items-center gap-3 p-3 bg-muted/20 border border-border/50 rounded-xl text-[10px] font-mono group/path overflow-hidden">
                <span class="px-2 py-0.5 rounded bg-amber-500/10 text-amber-600 border border-amber-500/20 font-black shrink-0 uppercase">Cloud</span>
                <span class="text-muted-foreground/80 truncate flex-1 hover:text-amber-600 transition-colors cursor-default" title={audioState.cloudDriveLocation}>
                    {audioState.cloudDriveLocation}
                </span>
            </div>
        </div>

        <div class="space-y-3">
            {#if fileState.recordedFiles.length === 0}
                <div class="py-12 text-center border-2 border-dashed border-border rounded-xl">
                    <p class="text-muted-foreground text-sm">No recordings found.</p>
                </div>
            {:else}
                {#each fileState.recordedFiles as file}
                    <div class="group flex flex-col sm:flex-row sm:items-center justify-between p-4 bg-muted/20 border border-border/50 rounded-xl hover:bg-muted/40 transition-all duration-200 gap-4">
                        <div class="flex items-center gap-4 flex-1 min-w-0">
                            <div class="min-w-0 w-full">
                                <h3 class="text-sm font-semibold text-card-foreground/90 truncate">{file.name}</h3>
                                <div class="flex flex-wrap gap-x-3 gap-y-1 mt-1 text-xs">
                                    <span class="text-muted-foreground font-medium uppercase tracking-wider">{formatSize(file.size)}</span>
                                    <span class="text-muted-foreground/40">•</span>
                                    <span class="text-muted-foreground">{formatDate(file.modTime)}</span>
                                </div>
                            </div>
                        </div>

                        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 w-full sm:w-auto">
                            <audio 
                                controls 
                                src="/api/recordings/{file.name}" 
                                class="h-8 w-full sm:w-[400px]"
                            ></audio>
                            <Button 
                                variant="ghost" 
                                size="sm" 
                                class="h-8 px-4 text-xs bg-muted/50 hover:bg-amber-500/10 hover:text-amber-500 border border-border whitespace-nowrap"
                                onclick={() => openPushDialog(file)}
                            >
                                <Cloud class="w-3.5 h-3.5 mr-2" />
                                Cloud Push
                            </Button>
                        </div>
                    </div>
                {/each}
            {/if}
        </div>
    </Card.Content>
</Card.Root>

<!-- Push Dialog Modal -->
{#if pushingFile}
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm animate-in fade-in duration-200">
        <div class="w-full max-w-md bg-card border border-border rounded-2xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
            <div class="p-6 space-y-6">
                <div class="flex items-center justify-between">
                    <h2 class="text-xl font-bold text-foreground flex items-center gap-2">
                        <Cloud class="w-5 h-5 text-amber-500" />
                        Push to Cloud
                    </h2>
                    <button class="text-muted-foreground hover:text-foreground transition-colors" onclick={() => pushingFile = null}>
                        <X class="w-5 h-5" />
                    </button>
                </div>

                <div class="p-4 bg-muted/30 rounded-xl border border-border/50 space-y-3">
                    <div class="flex items-start gap-3">
                        <Info class="w-4 h-4 text-primary mt-0.5" />
                        <div class="text-s text-muted-foreground leading-relaxed">
                            You are about to copy <span class="text-foreground font-semibold">{pushingFile.name}</span> to your configured cloud drive.
                        </div>
                    </div>
                </div>

                <div class="space-y-4">
                    <div class="space-y-2">
                        <Label for="targetName" class="text-sm text-muted-foreground">Target Filename</Label>
                        <Input 
                            id="targetName"
                            bind:value={targetFilename}
                            placeholder="my-recording.wav"
                            class="bg-muted/50 border-border text-foreground h-11"
                        />
                        <p class="text-[10px] text-muted-foreground mt-1 italic">
                            Destination: {fileState.cloudDriveLocation}
                        </p>
                    </div>
                </div>

                <div class="flex gap-3 pt-2">
                    <Button 
                        variant="ghost" 
                        class="flex-1 bg-secondary hover:bg-secondary/80 text-secondary-foreground h-11"
                        onclick={() => pushingFile = null}
                    >
                        Cancel
                    </Button>
                    <Button 
                        class="flex-1 bg-amber-600 hover:bg-amber-700 text-white font-bold h-11 shadow-lg shadow-amber-900/20"
                        onclick={handlePush}
                    >
                        Push Now
                    </Button>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    /* Custom audio player styling to match theme better */
    audio {
        color-scheme: dark;
    }
</style>
