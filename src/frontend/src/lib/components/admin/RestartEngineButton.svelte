<script lang="ts">
    import Button from "../ui/Button.svelte";
    import { RotateCw } from "lucide-svelte";

    let {
        isRecording,
        isRestarting,
        onrestart,
    }: { isRecording: boolean; isRestarting: boolean; onrestart: () => void } = $props();
</script>

<div class="w-full space-y-2">
    <Button
        variant="outline"
        class="w-full gap-2 font-bold"
        onclick={onrestart}
        disabled={isRecording || isRestarting}
    >
        <RotateCw class="w-4 h-4 {isRestarting ? 'animate-spin' : ''}" />
        {isRestarting ? "Restarting Engine..." : "Restart Engine"}
    </Button>
    <p class="text-xxs text-center text-muted-foreground">
        {#if isRecording}
            Stop recording to restart the engine.
        {:else}
            Re-scans audio devices and reloads settings.
        {/if}
    </p>
</div>
