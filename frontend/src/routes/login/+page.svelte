<script lang="ts">
    import LoginView from "$lib/components/LoginView.svelte";
    import { getAppContext } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";

    const { system } = getAppContext();

    $effect(() => {
        if (system.isAuthenticated) {
            goto("/admin");
        }
    });
</script>

<LoginView />
