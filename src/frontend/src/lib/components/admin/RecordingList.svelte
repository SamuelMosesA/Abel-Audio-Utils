<script lang="ts">
    import { Cloud, Scissors, RotateCcw, X, Upload } from "lucide-svelte";
    import Card from "../ui/Card.svelte";
    import Button from "../ui/Button.svelte";
    import { onMount } from "svelte";
    import { getAppContext, type RecordedFile } from "../../audioState.svelte";
    import AudioPlayer from "./AudioPlayer.svelte";
    import { formatTimecode, parseTimecode } from "../../utils/timecode";

    const { files, ui, audio } = getAppContext();
    let picker = $state<HTMLInputElement | null>(null);
    let dragging = $state(false);
    let uploading = $state(false);
    let uploadName = $state("");
    let uploadProgress = $state(0);
    let editing = $state<RecordedFile | null>(null);
    let previewPlayer = $state<AudioPlayer | null>(null);
    let previewPosition = $state(0);
    let startText = $state("0:00");
    let endText = $state("0:00");
    let busy = $state(false);
    let refreshing = $state(false);
    let pushing = $state("");
    let stopping = $state("");
    let error = $state("");

    const size = (bytes: number) => (bytes / 1048576).toFixed(1) + " MB";
    const time = formatTimecode;
    const active = (file: RecordedFile) => file.jobs.find(j => j.stage === "queued" || j.stage === "analyzing" || j.stage === "encoding" || j.stage === "pushing");
    const lastAutomatic = (file: RecordedFile) => [...file.jobs].reverse().find(j => j.autoPush);
    const processed = (file: RecordedFile) => file.exports.find(e => e.name.endsWith("-processed.mp3"));

    onMount(() => {
        void files.fetchFiles();
    });

    async function handleRefresh() {
        if (refreshing) return;
        refreshing = true;
        try {
            await files.fetchFiles();
        } finally {
            refreshing = false;
        }
    }

    async function uploadFiles(selected: File[]) {
        if (uploading || selected.length === 0) return;
        error = "";
        uploading = true;
        try {
            for (const file of selected) {
                if (!/\.(wav|mp3|m4a|flac|aac|ogg)$/i.test(file.name)) {
                    error = "Unsupported audio file: " + file.name;
                    continue;
                }
                uploadName = file.name;
                uploadProgress = 0;
                const result = await files.upload(file, percent => uploadProgress = percent);
                if (!result.success) {
                    error = file.name + ": " + (result.error || "Upload failed");
                } else if (result.processingError) {
                    error = file.name + " was imported, but processing could not start: " + result.processingError;
                } else {
                    ui.showNotification(file.name + " imported. Processing started.", "Recording import");
                }
            }
        } finally {
            uploading = false;
            uploadName = "";
        }
    }

    function drop(event: DragEvent) {
        event.preventDefault();
        dragging = false;
        void uploadFiles(Array.from(event.dataTransfer?.files || []));
    }

    function paste(event: ClipboardEvent) {
        const pasted = Array.from(event.clipboardData?.files || []);
        if (pasted.length === 0) {
            for (const item of Array.from(event.clipboardData?.items || [])) {
                const file = item.getAsFile();
                if (file) pasted.push(file);
            }
        }
        const mimeExtensions: Record<string, string> = {
            "audio/wav": ".wav", "audio/x-wav": ".wav", "audio/mpeg": ".mp3",
            "audio/mp4": ".m4a", "audio/flac": ".flac", "audio/aac": ".aac", "audio/ogg": ".ogg"
        };
        const audio = pasted.map(file => {
            if (/\.(wav|mp3|m4a|flac|aac|ogg)$/i.test(file.name)) return file;
            const extension = mimeExtensions[file.type];
            return extension ? new File([file], "clipboard-" + Date.now() + extension, { type: file.type }) : file;
        });
        if (audio.length > 0) {
            event.preventDefault();
            void uploadFiles(audio);
        }
    }

    function edit(file: RecordedFile) {
        const ready = processed(file);
        if (!ready || ready.duration < 0.5) return;
        editing = file;
        previewPosition = 0;
        startText = "0:00";
        endText = time(ready.duration);
        error = "";
    }

    async function trim() {
        if (!editing) return;
        const ready = processed(editing);
        const start = parseTimecode(startText);
        const end = parseTimecode(endText);
        if (!ready || start === null || end === null || start < 0 || end > ready.duration + 0.05 || end - start < 0.5) {
            error = "Use mm:ss or hh:mm:ss and keep at least 0.5 seconds within the processed audio.";
            return;
        }
        busy = true;
        const result = await files.process(editing.name, start, end);
        busy = false;
        if (result.success) {
            editing = null;
            ui.showNotification("Trim started. The MP3 will be pushed to cloud when ready.", "Recording trim");
        } else error = result.error || "Could not start trimming.";
    }

    async function retry(file: RecordedFile) {
        error = "";
        const result = await files.process(file.name, 0, 0, true);
        if (!result.success) error = result.error || "Could not start processing.";
    }

    async function push(name: string) {
        error = "";
        pushing = name;
        const result = await files.pushToCloud(name);
        pushing = "";
        if (!result.success) error = result.error || "Could not push audio to cloud.";
    }
    async function stop(id: string) {
        error = "";
        stopping = id;
        const result = await files.cancelProcessing(id);
        stopping = "";
        if (!result.success) error = result.error || "Could not stop processing.";
    }
    function setBoundary(boundary: "start" | "end") {
        if (boundary === "start") startText = time(previewPosition);
        else endText = time(previewPosition);
        error = "";
    }
    function previewSelection() {
        const start = parseTimecode(startText);
        const end = parseTimecode(endText);
        if (start === null || end === null || end - start < 0.5) { error = "Enter a valid start and end before previewing."; return; }
        void previewPlayer?.playSelection(start, end);
    }
</script>

<svelte:window onpaste={paste} />

<Card 
    title="Master Recording Library"
    description="Inspect audio captures, trim recordings, and manage automatic cloud push."
>
    <div class="space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-2 pb-2 border-b border-border/40">
            <div class="space-y-0.5">
                {#if audio.storageLocation}<p class="text-xs text-muted-foreground break-all">Recordings folder: <span class="text-foreground">{audio.storageLocation}</span></p>{/if}
                {#if audio.cloudDriveLocation}<p class="text-xs text-muted-foreground break-all">Cloud folder: <span class="text-foreground">{audio.cloudDriveLocation}</span></p>{/if}
            </div>
            <Button size="sm" variant="outline" onclick={handleRefresh} disabled={refreshing} title="Refresh recording library">
                <RotateCcw class="w-3.5 h-3.5 mr-1.5 {refreshing ? 'animate-spin' : ''}" />
                {refreshing ? "Refreshing..." : "Refresh"}
            </Button>
        </div>
        
        <div role="region" aria-label="Import audio" class="rounded-lg border-2 border-dashed p-6 text-center transition-colors {dragging ? 'border-primary bg-primary/5' : 'border-border/80 bg-muted/15'}"
            ondragenter={(event) => { event.preventDefault(); dragging = true; }}
            ondragover={(event) => event.preventDefault()}
            ondragleave={(event) => { if (!event.currentTarget.contains(event.relatedTarget as Node)) dragging = false; }}
            ondrop={drop}>
            <input bind:this={picker} type="file" accept="audio/*,.wav,.mp3,.m4a,.flac,.aac,.ogg" multiple class="sr-only"
                onchange={(event) => { void uploadFiles(Array.from(event.currentTarget.files || [])); event.currentTarget.value = ""; }} />
            <p class="text-sm font-medium mb-1">Import audio to process</p>
            <p class="text-xs text-muted-foreground mb-3">Choose files, drop them here, or paste from clipboard (WAV, MP3, M4A, FLAC, AAC, OGG).</p>
            <Button size="sm" variant="outline" onclick={() => picker?.click()} disabled={uploading}>
                <Upload class="w-3.5 h-3.5 mr-1.5" /> Choose files
            </Button>
            {#if uploading}
                <div class="mt-4 space-y-1.5" aria-live="polite">
                    <p class="text-xs text-muted-foreground">Uploading {uploadName} · {uploadProgress}%</p>
                    <progress class="w-full h-1.5 accent-primary bg-secondary rounded" value={uploadProgress} max="100"></progress>
                </div>
            {/if}
        </div>

        {#if error}<p role="alert" class="p-3 rounded-md bg-destructive/10 border border-destructive/20 text-destructive text-sm">{error}</p>{/if}
        {#if files.loadError}<p role="alert" class="p-3 rounded-md bg-destructive/10 border border-destructive/20 text-destructive text-sm">{files.loadError}</p>{/if}
        {#if files.recordedFiles.length === 0}
            <p class="py-10 text-center text-xs text-muted-foreground">No recordings captured yet</p>
        {/if}
        {#each files.recordedFiles as file (file.name)}
            {@const job = active(file)}
            {@const latest = lastAutomatic(file)}
            {@const failed = latest?.stage === "failed" ? latest : null}
            {@const initial = processed(file)}
            <section class="p-4 bg-muted/15 border border-border rounded-lg space-y-3">
                <div class="flex items-center justify-between gap-3">
                    <div class="min-w-0">
                        <h4 class="font-medium text-sm text-foreground truncate">{file.display || file.name.replace(/\.wav$/i, "")}</h4>
                        <p class="text-xs text-muted-foreground">{new Date(file.modTime).toLocaleString()} · {time(initial?.duration ?? file.duration)} · {size(initial?.size ?? file.size)}</p>
                        <p class="text-xs text-muted-foreground/80 break-all" title={audio.storageLocation + "/" + (initial?.name ?? file.name)}>{audio.storageLocation}/{initial?.name ?? file.name}</p>
                    </div>
                    <Button size="sm" variant="outline" onclick={() => edit(file)} disabled={!initial || initial.duration < 0.5} title={!initial ? "Available after MP3 processing finishes" : "Trim processed MP3"}>
                        <Scissors class="w-3.5 h-3.5 mr-1.5" /> Trim
                    </Button>
                </div>
                {#if job}
                    <div class="space-y-1.5" aria-live="polite">
                        <div class="flex items-center justify-between gap-2 text-xs capitalize text-muted-foreground">
                            <span>{job.stage} audio…</span>
                            <span>{(job.stage === "analyzing" || job.stage === "encoding") && job.totalSeconds ? `${time(job.processedSeconds || 0)} of ${time(job.totalSeconds)}` : ""}</span>
                            <Button size="sm" variant="outline" onclick={() => stop(job.id)} disabled={stopping === job.id}>{stopping === job.id ? "Stopping…" : "Stop"}</Button>
                        </div>
                        <progress class="w-full h-1.5 accent-primary bg-secondary rounded" value={(job.stage === "analyzing" || job.stage === "encoding") ? job.progress : undefined} max="100"></progress>
                    </div>
                {:else if !initial}
                    <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground">
                        <span>{file.duration <= 0 ? "Recording is still being captured or has no audio." : failed ? "Processing failed: " + failed.error : latest?.stage === "cancelled" ? "Processing stopped." : "Processing has not started."}</span>
                        <Button size="sm" variant="outline" onclick={() => retry(file)} disabled={file.duration <= 0}>
                            <RotateCcw class="w-3.5 h-3.5 mr-1.5" /> Retry
                        </Button>
                    </div>
                    {#if file.duration > 0}
                        <div class="space-y-2 rounded-md bg-background border border-border/60 p-3">
                            <p class="text-xs text-muted-foreground">Original audio · {file.name}</p>
                            <AudioPlayer src="/api/recordings/raw/{encodeURIComponent(file.name)}" duration={file.duration} />
                            {#if failed && file.name.toLowerCase().endsWith(".wav")}
                                <p class="text-xs text-muted-foreground break-all">{file.rawPushed ? "Pushed to cloud" : "Cloud destination"}: {file.rawCloudPath}</p>
                                {#if !file.rawPushed}
                                    <Button size="sm" variant="outline" onclick={() => push(file.name)} disabled={pushing === file.name}>
                                        <Cloud class="w-3.5 h-3.5 mr-1.5" /> {pushing === file.name ? "Pushing…" : "Push WAV to cloud"}
                                    </Button>
                                {/if}
                            {/if}
                        </div>
                    {/if}
                {:else if failed && failed.autoPush && !initial.pushed}
                    <p role="alert" class="text-xs text-destructive">Cloud push failed: {failed.error}. Check cloud folder and logs.</p>
                {/if}
                {#each file.jobs.filter(j => !j.autoPush && j.stage === "failed") as failedTrim (failedTrim.id)}
                    <p role="alert" class="text-xs text-destructive">Trim or push failed: {failedTrim.error}</p>
                {/each}
                {#each file.jobs.filter(j => !j.autoPush && j.stage === "cancelled") as stoppedTrim (stoppedTrim.id)}
                    <p class="text-xs text-muted-foreground">Trim stopped.</p>
                {/each}
                {#each file.exports as exportFile (exportFile.name)}
                    <div class="flex flex-col xl:flex-row xl:items-center gap-3 rounded-md bg-background border border-border/60 p-3">
                        <div class="min-w-0 flex-1">
                            <p class="text-xs font-semibold text-foreground">{exportFile.name === initial?.name ? "Processed MP3" : "Trimmed MP3"}</p>
                            <p class="text-xs text-muted-foreground">{size(exportFile.size)} · {exportFile.pushed ? "Pushed to cloud" : job?.output === exportFile.name && job.stage === "pushing" ? "Pushing to cloud" : "Pending automatic push"}</p>
                            <p class="text-xs text-muted-foreground/80 break-all" title={audio.storageLocation + "/" + exportFile.name}>Local: {audio.storageLocation}/{exportFile.name}</p>
                            <p class="text-xs text-muted-foreground/80 break-all" title={exportFile.cloudPath}>{exportFile.pushed ? "Cloud" : "Destination"}: {exportFile.cloudPath}</p>
                        </div>
                        <div class="w-full xl:w-64">
                            <AudioPlayer src="/api/recordings/raw/{encodeURIComponent(exportFile.name)}" duration={exportFile.duration} />
                        </div>
                    </div>
                {/each}
            </section>
        {/each}
    </div>
</Card>

{#if editing}
    {@const ready = processed(editing)}
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
        <div class="w-full max-w-xl bg-card border border-border rounded-lg shadow-lg p-6 space-y-4">
            <div class="flex items-center justify-between pb-3 border-b border-border/60">
                <div>
                    <h3 class="text-base font-semibold text-foreground">Trim Recording</h3>
                    <p class="text-xs text-muted-foreground">Set audio boundaries to generate a trimmed MP3 export.</p>
                </div>
                <Button variant="ghost" size="icon" aria-label="Close trim editor" onclick={() => editing = null}>
                    <X class="w-4 h-4" />
                </Button>
            </div>
            {#if ready}
                <p class="text-xs text-muted-foreground">Target: {ready.name} · {time(ready.duration)}</p>
                <div class="p-2 bg-muted/20 rounded-md border border-border/40">
                    <AudioPlayer bind:this={previewPlayer} bind:position={previewPosition} src="/api/recordings/raw/{encodeURIComponent(ready.name)}" duration={ready.duration} selectionEnd={parseTimecode(endText) ?? undefined} />
                </div>
            {/if}
            <div class="grid grid-cols-2 gap-4">
                <div class="space-y-1.5">
                    <label for="trim-start" class="text-xs font-medium text-muted-foreground">Start (mm:ss)</label>
                    <input id="trim-start" type="text" inputmode="numeric" bind:value={startText} class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring" />
                    <Button variant="outline" size="sm" class="w-full mt-1" onclick={() => setBoundary("start")}>Set to playhead</Button>
                </div>
                <div class="space-y-1.5">
                    <label for="trim-end" class="text-xs font-medium text-muted-foreground">End (mm:ss)</label>
                    <input id="trim-end" type="text" inputmode="numeric" bind:value={endText} class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring" />
                    <Button variant="outline" size="sm" class="w-full mt-1" onclick={() => setBoundary("end")}>Set to playhead</Button>
                </div>
            </div>
            <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground py-1">
                <span>Export duration: {parseTimecode(startText) !== null && parseTimecode(endText) !== null ? time(Math.max(0, (parseTimecode(endText) ?? 0) - (parseTimecode(startText) ?? 0))) : "—"}</span>
                <div class="flex gap-2">
                    <Button variant="ghost" size="sm" onclick={() => { startText = "0:00"; endText = time(processed(editing!)?.duration ?? 0); }}>Reset</Button>
                    <Button variant="outline" size="sm" onclick={previewSelection}>Play selection</Button>
                </div>
            </div>
            {#if error}<p role="alert" class="text-xs text-destructive">{error}</p>{/if}
            <div class="flex justify-end gap-2 pt-2 border-t border-border/60">
                <Button variant="ghost" size="sm" onclick={() => editing = null}>Cancel</Button>
                <Button size="sm" onclick={trim} disabled={busy}>{busy ? "Processing…" : "Create & Push MP3"}</Button>
            </div>
        </div>
    </div>
{/if}
