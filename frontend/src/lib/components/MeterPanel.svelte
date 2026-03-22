<script lang="ts">
    import { getAppContext } from "../audioState.svelte";
    const { visuals } = getAppContext();
    import { cn } from "$lib/utils/utils";
</script>

<div class="space-y-10">
    <div class="grid grid-cols-1 gap-12">
        <!-- Left -->
        <div class="space-y-4">
            <div class="flex items-center justify-between text-xs font-black tracking-[0.2em] text-muted-foreground uppercase">
                <div class="flex items-center gap-3">
                    <div class="w-1.5 h-3 bg-primary rounded-full"></div>
                    System L
                </div>
                <div class={cn("font-mono text-base font-black transition-colors duration-200", visuals.currentDb.L > -3 ? "text-destructive" : "text-primary")}>
                    {visuals.currentDb.L <= -99 ? "−∞" : `${visuals.currentDb.L.toFixed(1)} dB`}
                </div>
            </div>
            <div class="h-6 bg-black border border-border/60 rounded-xl overflow-hidden p-1 relative shadow-inner">
                <div
                    class={cn(
                        "h-full transition-all duration-75 rounded-lg relative overflow-hidden",
                        visuals.currentDb.L > -12 
                            ? "bg-gradient-to-r from-emerald-500 from-60% via-yellow-400 via-85% to-destructive" 
                            : "bg-primary"
                    )}
                    style="width: {visuals.currentMeters.L}%"
                >
                    <div class="absolute inset-0 opacity-20 bg-[linear-gradient(90deg,transparent_90%,theme(colors.black)_90%)] bg-[length: theme(spacing.5)_100%]"></div>
                </div>
            </div>
        </div>

        <!-- Right -->
        <div class="space-y-4">
            <div class="flex items-center justify-between text-xs font-black tracking-[0.2em] text-muted-foreground uppercase">
                <div class="flex items-center gap-3">
                    <div class="w-1.5 h-3 bg-primary rounded-full"></div>
                    System R
                </div>
                <div class={cn("font-mono text-base font-black transition-colors duration-200", visuals.currentDb.R > -3 ? "text-destructive" : "text-primary")}>
                    {visuals.currentDb.R <= -99 ? "−∞" : `${visuals.currentDb.R.toFixed(1)} dB`}
                </div>
            </div>
            <div class="h-6 bg-black border border-border/60 rounded-xl overflow-hidden p-1 relative shadow-inner">
                <div
                    class={cn(
                        "h-full transition-all duration-75 rounded-lg relative overflow-hidden",
                        visuals.currentDb.R > -12 
                            ? "bg-gradient-to-r from-emerald-500 from-60% via-yellow-400 via-85% to-destructive" 
                            : "bg-primary"
                    )}
                    style="width: {visuals.currentMeters.R}%"
                >
                    <div class="absolute inset-0 opacity-20 bg-[linear-gradient(90deg,transparent_90%,theme(colors.black)_90%)] bg-[length: theme(spacing.5)_100%]"></div>
                </div>
            </div>
        </div>
    </div>
</div>
