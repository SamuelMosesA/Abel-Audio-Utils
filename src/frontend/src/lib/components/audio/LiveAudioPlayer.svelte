<script lang="ts">
    import { Play, RotateCcw } from "lucide-svelte";
    import Button from "../ui/Button.svelte";

    let { src, label = "Live audio" } = $props<{ src: string; label?: string }>();

    let audioElement = $state<HTMLAudioElement | null>(null);
    let errorMessage = $state("");
    let isStarting = $state(false);
    let isPlaying = $state(false);

    async function startPlayback() {
        if (!audioElement || isStarting) return;
        errorMessage = "";
        isStarting = true;
        try {
            await audioElement.play();
            isPlaying = true;
        } catch {
            errorMessage = "Browser blocked playback. Tap Retry to start audio.";
        } finally {
            isStarting = false;
        }
    }
</script>

<div class="space-y-3">
    <audio
        bind:this={audioElement}
        controls
        playsinline
        preload="none"
        {src}
        aria-label={label}
        onplaying={() => {
            isPlaying = true;
            errorMessage = "";
        }}
        onerror={() => {
            errorMessage = "Live audio stream unavailable. Tap Retry to reconnect.";
        }}
        class="w-full h-10 rounded-lg opacity-85 hover:opacity-100 transition-opacity"
    ></audio>

    {#if !isPlaying || errorMessage}
        <Button variant="secondary" size="sm" onclick={startPlayback} disabled={isStarting} class="w-full sm:w-auto">
            {#if errorMessage}
                <RotateCcw class="w-4 h-4 mr-2" /> Retry Listening
            {:else}
                <Play class="w-4 h-4 mr-2" /> {isStarting ? "Starting…" : "Start Listening"}
            {/if}
        </Button>
    {/if}

    {#if errorMessage}
        <p role="alert" class="text-xs text-destructive">{errorMessage}</p>
    {/if}
</div>
