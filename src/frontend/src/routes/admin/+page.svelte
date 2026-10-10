<script lang="ts">
    import AudioAdminView from "$lib/components/admin/AudioAdminView.svelte";
    import { getAppContext } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";

    const { system, ui } = getAppContext();

    // Reactive check for auth changes (logout or initial unauthenticated visit)
    $effect(() => {
        if (!system.isAuthenticated) {
            goto("/login?redirect=/admin");
        }
    });
</script>

{#if system.isAuthenticated}
    <AudioAdminView />
{/if}
