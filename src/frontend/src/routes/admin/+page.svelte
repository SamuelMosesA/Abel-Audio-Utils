<script lang="ts">
    import AudioAdminView from "$lib/components/admin/AudioAdminView.svelte";
    import { getAppContext } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";

    const { system } = getAppContext();

    // Reactive check for auth changes (logout or unauthenticated)
    $effect(() => {
        if (system.authChecked && !system.isAuthenticated) {
            goto("/login");
        }
    });
</script>

{#if system.authChecked && system.isAuthenticated}
    <AudioAdminView />
{/if}
