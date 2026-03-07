<script lang="ts">
    import AudioAdminView from "$lib/components/AudioAdminView.svelte";
    import { audioState } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";
    import { onMount } from "svelte";

    onMount(() => {
        audioState.currentView = "admin";
        if (!audioState.isAuthenticated) {
            goto("/login");
        } else {
            audioState.syncStatus();
        }
    });

    // Reactive check for auth changes (logout)
    $effect(() => {
        if (!audioState.isAuthenticated) {
            goto("/login");
        }
    });
</script>

{#if audioState.isAuthenticated}
    <AudioAdminView />
{/if}
