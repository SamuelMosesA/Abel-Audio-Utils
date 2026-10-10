<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    import { goto } from "$app/navigation";
    import type { Snippet } from "svelte";

    let { children }: { children?: Snippet } = $props();
    const { system } = getAppContext();

    let isChecking = $state(true);

    $effect(() => {
        let active = true;
        (async () => {
            const valid = await system.validateSession();
            if (!active) return;
            isChecking = false;
            if (!valid) {
                goto("/login?redirect=/admin");
            }
        })();
        return () => {
            active = false;
        };
    });
</script>

{#if isChecking}
    <div class="flex items-center justify-center min-h-[60vh] text-slate-400">
        <div class="flex flex-col items-center gap-3">
            <div class="w-8 h-8 border-3 border-indigo-500 border-t-transparent rounded-full animate-spin"></div>
            <span class="text-sm">Verifying access...</span>
        </div>
    </div>
{:else if system.isAuthenticated}
    {@render children?.()}
{/if}
