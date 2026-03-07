<script lang="ts">
    import LoginView from "$lib/components/LoginView.svelte";
    import { audioState } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";
    import { onMount } from "svelte";

    onMount(() => {
        if (audioState.isAuthenticated) {
            goto("/admin");
        }
    });
</script>

<LoginView />
