<script lang="ts">
    import StreamingView from "$lib/components/StreamingView.svelte";
    import { onMount, onDestroy } from "svelte";
    import { audioState } from "$lib/audioState.svelte";

    onMount(() => {
        audioState.currentView = "stream";
    });
</script>

<StreamingView />
