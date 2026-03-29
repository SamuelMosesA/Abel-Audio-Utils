<script lang="ts">
    import AudioAdminView from "$lib/components/admin/AudioAdminView.svelte";
    import { getAppContext } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";

    const { system, ui } = getAppContext();

    // Reactive check for auth changes (logout)
    $effect(() => {
        if (!system.isAuthenticated) {
            goto("/login");
        }
    });
</script>

{#if system.isAuthenticated}
    <AudioAdminView />
{/if}
