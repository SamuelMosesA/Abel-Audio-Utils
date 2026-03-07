<script lang="ts">
    import { audioVisuals } from "../audioVisuals.svelte";
    import { cn } from "$lib/utils/utils";
</script>

<div class="space-y-6">
    <div class="flex justify-between items-end px-1">
        <span class="text-xs font-bold text-muted-foreground uppercase tracking-widest">Peak Meters</span>
    </div>

    <div class="space-y-5">
        <!-- Left -->
        <div class="space-y-2">
            <div class="flex items-center justify-between text-s font-black font-mono text-muted-foreground/80 tracking-tighter">
                <span class="flex items-center gap-1.5">
                    <span class="w-1 h-3 bg-primary rounded-full"></span>
                    LEFT CHANNEL
                </span>
                <span class={cn("transition-colors", audioVisuals.currentDb.L > -3 ? "text-red-500 font-bold" : "text-primary/90")}>
                    {audioVisuals.currentDb.L <= -99 ? "−∞" : `${audioVisuals.currentDb.L.toFixed(1)} dB`}
                </span>
            </div>
            <div class="h-3 bg-black/20 rounded-full overflow-hidden border border-white/5 shadow-inner p-0.5">
                <div
                    class={cn(
                        "h-full transition-all duration-75 rounded-full shadow-[0_0_15px_hsl(var(--primary)/0.4)] min-w-[2px]",
                        audioVisuals.currentDb.L > -3
                            ? "bg-gradient-to-r from-emerald-500 via-yellow-400 to-red-600 shadow-red-500/20"
                            : "bg-primary",
                    )}
                    style="width: {audioVisuals.currentMeters.L}%"
                ></div>
            </div>
        </div>

        <!-- Right -->
        <div class="space-y-2">
            <div class="flex items-center justify-between text-s font-black font-mono text-muted-foreground/80 tracking-tighter">
                <span class="flex items-center gap-1.5">
                    <span class="w-1 h-3 bg-red-500 rounded-full"></span>
                    RIGHT CHANNEL
                </span>
                <span class={cn("transition-colors", audioVisuals.currentDb.R > -3 ? "text-red-500 font-bold" : "text-primary/90")}>
                    {audioVisuals.currentDb.R <= -99 ? "−∞" : `${audioVisuals.currentDb.R.toFixed(1)} dB`}
                </span>
            </div>
            <div class="h-3 bg-black/20 rounded-full overflow-hidden border border-white/5 shadow-inner p-0.5">
                <div
                    class={cn(
                        "h-full transition-all duration-75 rounded-full shadow-[0_0_15px_hsl(var(--primary)/0.4)] min-w-[2px]",
                        audioVisuals.currentDb.R > -3
                            ? "bg-gradient-to-r from-emerald-500 via-yellow-400 to-red-600 shadow-red-500/20"
                            : "bg-primary",
                    )}
                    style="width: {audioVisuals.currentMeters.R}%"
                ></div>
            </div>
        </div>
    </div>
</div>
