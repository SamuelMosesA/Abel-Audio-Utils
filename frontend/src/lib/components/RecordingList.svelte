<script lang="ts">
    import { fileState, type RecordedFile } from "../fileState.svelte";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import { Input } from "$lib/components/ui/input/index.js";
    import { Label } from "$lib/components/ui/label/index.js";
    import { Play, Download, Cloud, X, Info } from "lucide-svelte";
    import { cn } from "$lib/utils";

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

<Card.Root class="bg-slate-900/40 border-slate-800 backdrop-blur-xl shadow-2xl">
    <Card.Header>
        <Card.Title class="flex items-center gap-2 text-slate-300 text-lg">
            <Download class="w-5 h-5 text-indigo-400" />
            Recorded Files
        </Card.Title>
        <Card.Description class="text-slate-500 text-xs">
            Manage and playback your recordings. Push to cloud drive for sharing.
        </Card.Description>
    </Card.Header>
    <Card.Content>
        <div class="space-y-3">
            {#if fileState.recordedFiles.length === 0}
                <div class="py-12 text-center border-2 border-dashed border-slate-800 rounded-xl">
                    <p class="text-slate-600 text-sm">No recordings found.</p>
                </div>
            {:else}
                {#each fileState.recordedFiles as file}
                    <div class="group flex items-center justify-between p-4 bg-slate-950/40 border border-slate-800/50 rounded-xl hover:bg-slate-900/60 transition-all duration-200">
                        <div class="flex items-center gap-4 flex-1">
                            <div class="min-w-0">
                                <h3 class="text-sm font-semibold text-slate-300 truncate">{file.name}</h3>
                                <div class="flex gap-3 mt-1">
                                    <span class="text-[10px] text-slate-500 font-medium uppercase tracking-wider">{formatSize(file.size)}</span>
                                    <span class="text-[10px] text-slate-600">•</span>
                                    <span class="text-[10px] text-slate-500">{formatDate(file.modTime)}</span>
                                </div>
                            </div>
                        </div>

                        <div class="flex items-center gap-2">
                            <audio 
                                controls 
                                src="/api/recordings/{file.name}" 
                                class="h-8 max-w-[150px] md:max-w-none opacity-50 hover:opacity-100 transition-opacity"
                            ></audio>
                            <Button 
                                variant="ghost" 
                                size="sm" 
                                class="h-8 px-3 text-xs bg-slate-900/50 hover:bg-amber-500/10 hover:text-amber-500 border border-slate-800"
                                onclick={() => openPushDialog(file)}
                            >
                                <Cloud class="w-3.5 h-3.5 mr-2" />
                                Cloud
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
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-in fade-in duration-200">
        <div class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
            <div class="p-6 space-y-6">
                <div class="flex items-center justify-between">
                    <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
                        <Cloud class="w-5 h-5 text-amber-500" />
                        Push to Cloud
                    </h2>
                    <button class="text-slate-500 hover:text-slate-300 transition-colors" onclick={() => pushingFile = null}>
                        <X class="w-5 h-5" />
                    </button>
                </div>

                <div class="p-4 bg-slate-950/50 rounded-xl border border-slate-800/50 space-y-3">
                    <div class="flex items-start gap-3">
                        <Info class="w-4 h-4 text-indigo-400 mt-0.5" />
                        <div class="text-xs text-slate-400 leading-relaxed">
                            You are about to copy <span class="text-slate-200 font-semibold">{pushingFile.name}</span> to your configured cloud drive.
                        </div>
                    </div>
                </div>

                <div class="space-y-4">
                    <div class="space-y-2">
                        <Label for="targetName" class="text-sm text-slate-400">Target Filename</Label>
                        <Input 
                            id="targetName"
                            bind:value={targetFilename}
                            placeholder="my-recording.wav"
                            class="bg-slate-950 border-slate-800 text-slate-100 h-11"
                        />
                        <p class="text-[10px] text-slate-500 mt-1 italic">
                            Destination: {fileState.cloudDriveLocation}
                        </p>
                    </div>
                </div>

                <div class="flex gap-3 pt-2">
                    <Button 
                        variant="ghost" 
                        class="flex-1 bg-slate-800 hover:bg-slate-700 text-slate-300 h-11"
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
    audio::-webkit-media-controls-enclosure {
        background-color: transparent;
    }
    audio::-webkit-media-controls-panel {
        background-color: rgba(30, 41, 59, 0.5);
    }
    /* Force text color for some browsers */
    audio::-webkit-media-controls-current-time-display,
    audio::-webkit-media-controls-time-remaining-display {
        color: #94a3b8; /* slate-400 */
        text-shadow: none;
    }
</style>
