<script lang="ts">
    import { FileAudio, Cloud, X, Info, HardDrive, Archive } from "lucide-svelte";
    import Card from "../ui/Card.svelte";
    import Button from "../ui/Button.svelte";
    import { getAppContext, type RecordedFile } from "../../audioState.svelte";

    const { files, audio } = getAppContext();

    let pushingFile = $state<RecordedFile | null>(null);
    let targetFilename = $state("");

    const openPushDialog = (file: RecordedFile) => {
        pushingFile = file;
        const date = new Date().toISOString().split('T')[0];
        targetFilename = `${date}.wav`;
    };

    const handlePush = async () => {
        if (!pushingFile || !targetFilename) return;
        const res = await files.pushToCloud(pushingFile.name, targetFilename);
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

<Card title="Master Recording Library">
    <div class="space-y-8">
        <!-- System Paths -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 pb-4">
            <div class="p-4 bg-muted/20 border border-dashed border-border rounded-xl space-y-2 group overflow-hidden">
                <div class="flex items-center gap-2 text-xxs font-black uppercase tracking-widest text-muted-foreground group-hover:text-primary transition-colors">
                    <HardDrive class="w-3 h-3" />
                    Storage Cluster
                </div>
                <p class="text-micro font-mono text-white/60 truncate" title={audio.storageLocation}>
                    {audio.storageLocation}
                </p>
            </div>
            <div class="p-4 bg-muted/20 border border-dashed border-border rounded-xl space-y-2 group overflow-hidden">
                <div class="flex items-center gap-2 text-xxs font-black uppercase tracking-widest text-muted-foreground group-hover:text-primary transition-colors">
                    <Archive class="w-3 h-3" />
                    Remote Archive
                </div>
                <p class="text-micro font-mono text-white/60 truncate" title={audio.cloudDriveLocation}>
                    {audio.cloudDriveLocation}
                </p>
            </div>
        </div>

        <div class="space-y-3">
            {#if files.recordedFiles.length === 0}
                <div class="py-16 text-center border-2 border-dashed border-border/20 rounded-2xl flex flex-col items-center gap-3">
                    <FileAudio class="w-10 h-10 text-muted/10" />
                    <p class="text-xs font-bold uppercase tracking-widest text-muted/40">No System Archives Found</p>
                </div>
            {:else}
                {#each files.recordedFiles as file}
                    <div class="flex flex-col xl:flex-row xl:items-center justify-between p-4 bg-muted/20 border border-border/40 rounded-xl hover:bg-muted/30 transition-all gap-4">
                        <div class="flex items-center gap-4 min-w-0">
                            <div class="p-3 bg-primary/10 rounded-xl text-primary shrink-0">
                                <FileAudio class="w-6 h-6" />
                            </div>
                            <div class="space-y-1 min-w-0">
                                <h3 class="font-bold text-sm text-white truncate">{file.name}</h3>
                                <div class="flex items-center gap-3 text-xxs font-black uppercase tracking-widest text-muted-foreground/60">
                                    <span class="text-primary/70">{formatSize(file.size)}</span>
                                    <span>•</span>
                                    <span>{formatDate(file.modTime)}</span>
                                </div>
                            </div>
                        </div>

                        <div class="flex items-center gap-4 w-full xl:w-auto">
                            <audio 
                                controls 
                                src="/api/recordings/raw/{file.name}" 
                                class="h-9 flex-1 xl:w-64"
                            ></audio>
                            <Button size="sm" variant="outline" onclick={() => openPushDialog(file)}>
                                <Cloud class="w-4 h-4 mr-2" /> Push
                            </Button>
                        </div>
                    </div>
                {/each}
            {/if}
        </div>
    </div>
</Card>

<!-- Push Dialog Modal -->
{#if pushingFile}
    <div class="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/90 backdrop-blur-sm animate-in fade-in duration-300">
        <Card title="Cloud Backup" class="w-full max-w-lg glass border-primary/20 p-2">
            <div class="flex items-center justify-between mb-8 p-4 border-b border-border/40">
                <div class="flex items-center gap-3">
                    <div class="p-2 bg-primary/20 rounded-lg">
                        <Cloud class="w-5 h-5 text-primary" />
                    </div>
                    <div class="flex flex-col">
                        <h2 class="text-lg font-bold text-white uppercase tracking-tight">Cloud Backup</h2>
                        <span class="text-xxs font-black uppercase tracking-widest text-muted-foreground">Transferring session</span>
                    </div>
                </div>
                <Button variant="ghost" size="icon" onclick={() => pushingFile = null}>
                    <X class="w-5 h-5" />
                </Button>
            </div>

            <div class="space-y-8 p-4">
                <div class="p-4 bg-primary/5 border border-primary/10 rounded-xl flex gap-3 text-xs text-muted-foreground leading-relaxed">
                    <Info class="w-5 h-5 text-primary shrink-0" />
                    <div>
                        Source File: <span class="text-white font-bold">{pushingFile.name}</span>
                    </div>
                </div>

                <div class="space-y-3">
                    <label for="archive-filename" class="text-xxs font-black uppercase tracking-widest text-muted-foreground ml-1">Archive Filename</label>
                    <input 
                        id="archive-filename"
                        bind:value={targetFilename}
                        class="w-full bg-muted/50 border border-border rounded-lg px-4 py-3 font-mono text-sm text-white focus:ring-primary"
                    />
                </div>

                <div class="flex gap-3 pt-4">
                    <Button 
                        variant="secondary" 
                        class="flex-1"
                        onclick={() => pushingFile = null}
                    >
                        Cancel
                    </Button>
                    <Button 
                        class="flex-1"
                        onclick={handlePush}
                    >
                        Execute Upload
                    </Button>
                </div>
            </div>
        </Card>
    </div>
{/if}

<style>
    audio {
        color-scheme: dark;
    }
</style>
