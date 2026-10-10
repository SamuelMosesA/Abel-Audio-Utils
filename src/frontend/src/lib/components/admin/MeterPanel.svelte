<script lang="ts">
    import { getAppContext } from "../../audioState.svelte";
    const { visuals } = getAppContext();
    import ActivityBar from "./ActivityBar.svelte";
    import { cn } from "$lib/utils/utils";
</script>

<div class="space-y-4">
    <div class="grid grid-cols-1 gap-4">
        <!-- Left -->
        <div class="space-y-1.5">
            <div class="flex justify-between text-xs font-medium text-muted-foreground">
                <span>Channel Left</span>
                <span class={cn("font-mono font-semibold", visuals.currentDb.L > -3 ? "text-destructive" : "text-primary")}>
                    {visuals.currentDb.L <= -99 ? "−∞" : `${visuals.currentDb.L.toFixed(1)} dB`}
                </span>
            </div>
            <ActivityBar value={visuals.currentMeters.L / 100} />
        </div>

        <!-- Right -->
        <div class="space-y-1.5">
            <div class="flex items-center justify-between text-xs font-medium text-muted-foreground">
                <span>Channel Right</span>
                <span class={cn("font-mono font-semibold", visuals.currentDb.R > -3 ? "text-destructive" : "text-primary")}>
                    {visuals.currentDb.R <= -99 ? "−∞" : `${visuals.currentDb.R.toFixed(1)} dB`}
                </span>
            </div>
            <ActivityBar value={visuals.currentMeters.R / 100} />
        </div>
    </div>
</div>
