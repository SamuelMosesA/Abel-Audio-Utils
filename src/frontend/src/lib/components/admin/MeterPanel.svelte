<script lang="ts">
    import { getAppContext } from "../../audioState.svelte";
    const { visuals } = getAppContext();
    import ActivityBar from "./ActivityBar.svelte";
    import { cn } from "$lib/utils/utils";
</script>

<div class="space-y-6">
    <div class="grid grid-cols-1 gap-6">
        <!-- Left -->
        <div class="space-y-2">
            <div class="flex justify-between text-xxs uppercase font-bold text-muted-foreground">
      <span>System Left</span>
      <span class={cn("font-mono font-black", visuals.currentDb.L > -3 ? "text-destructive" : "text-primary")}>
                    {visuals.currentDb.L <= -99 ? "−∞" : `${visuals.currentDb.L.toFixed(1)} dB`}
                </span>
    </div>
            <ActivityBar value={visuals.currentMeters.L / 100} />
        </div>

        <!-- Right -->
        <div class="space-y-2">
            <div class="flex items-center justify-between text-xxs font-black tracking-widest text-muted-foreground uppercase">
                <span>System Right</span>
                <span class={cn("font-mono font-black", visuals.currentDb.R > -3 ? "text-destructive" : "text-primary")}>
                    {visuals.currentDb.R <= -99 ? "−∞" : `${visuals.currentDb.R.toFixed(1)} dB`}
                </span>
            </div>
            <ActivityBar value={visuals.currentMeters.R / 100} />
        </div>
    </div>
</div>
