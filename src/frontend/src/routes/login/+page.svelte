<script lang="ts">
    import LoginView from "$lib/components/login/LoginView.svelte";
    import { getAppContext } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { resolveRedirect } from "$lib/utils/redirect";

    const { system } = getAppContext();

    $effect(() => {
        if (system.isAuthenticated) {
            goto(resolveRedirect(page.url.searchParams.get("redirect")));
        }
    });
</script>

<LoginView />
