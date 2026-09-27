<script lang="ts">
    import { Play, RotateCcw } from "lucide-svelte";
    import Button from "../ui/Button.svelte";

    let { src, label = "Live audio" } = $props<{ src: string; label?: string }>();

    let audioElement = $state<HTMLAudioElement | null>(null);
    let errorMessage = $state("");
    let isStarting = $state(false);
    let hasPlayed = $state(false);

    async function startPlayback() {
        if (!audioElement || isStarting) return;
        errorMessage = "";
        isStarting = true;
        try {
            await audioElement.play();
            hasPlayed = true;
        } catch {
            errorMessage = "Safari blocked playback or the live stream is not ready. Tap Retry to try again.";
        } finally {
            isStarting = false;
        }
    }

    function handleMediaError() {
        errorMessage = "The live audio stream could not be loaded. Tap Retry when the broadcast is active.";
    }
</script>

<div class="space-y-3">
    <audio
        bind:this={audioElement}
        controls
        preload="none"
        {src}
        aria-label={label}
        onerror={handleMediaError}
        onplaying={() => {
            hasPlayed = true;
            errorMessage = "";
        }}
        class="w-full h-10 rounded-lg opacity-80 hover:opacity-100 transition-opacity"
    ></audio>

    {#if !hasPlayed || errorMessage}
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
