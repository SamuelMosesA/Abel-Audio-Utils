<script lang="ts">
    import LandingView from "$lib/components/LandingView.svelte";
    import { audioState } from "$lib/audioState.svelte";
    import { onMount } from "svelte";

    onMount(() => {
        audioState.currentView = "landing";
    });
</script>

<LandingView />
