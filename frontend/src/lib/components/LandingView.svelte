<script lang="ts">
    import { goto } from "$app/navigation";
    import { audioState } from "$lib/audioState.svelte";
    import { Button } from "$lib/components/ui/button/index.js";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Mic, Headphones, ArrowRight, ShieldCheck } from "lucide-svelte";
</script>

<div class="h-[70vh] flex flex-col items-center justify-center space-y-12 animate-in fade-in zoom-in-95 duration-700">
    <div class="text-center space-y-4">
        <h1 class="text-5xl md:text-6xl font-black tracking-tighter bg-gradient-to-b from-foreground to-muted-foreground bg-clip-text text-transparent">
            X32 CONSOLE
        </h1>
        <p class="text-muted-foreground text-lg font-medium tracking-wide border-t border-border pt-2">
            PROFESSIONAL AUDIO ENGINE
        </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-8 w-full max-w-3xl">
        <!-- Listener Card -->
        <button 
            onclick={() => goto("/stream")}
            class="group relative"
        >
            <div class="absolute -inset-1 bg-gradient-to-r from-primary/20 to-emerald-500/20 rounded-3xl blur opacity-25 group-hover:opacity-100 transition duration-500"></div>
            <Card.Root class="relative bg-card/40 border-border backdrop-blur-xl h-full transition-all duration-300 group-hover:bg-card/60 group-hover:border-primary/50 text-left overflow-hidden">
                <Card.Header>
                    <div class="p-3 bg-muted rounded-2xl w-fit group-hover:bg-primary group-hover:text-primary-foreground transition-colors duration-300">
                        <Headphones class="w-6 h-6" />
                    </div>
                </Card.Header>
                <Card.Content class="space-y-2">
                    <Card.Title class="text-2xl">Listen to Stream</Card.Title>
                    <Card.Description class="text-base">Join as a listener to hear the live audio feed from the console.</Card.Description>
                </Card.Content>
                <Card.Footer>
                    <div class="flex items-center text-sm font-bold text-primary opacity-0 group-hover:opacity-100 transition-all duration-300 translate-x-[-10px] group-hover:translate-x-0">
                        START STREAMING <ArrowRight class="ml-2 w-4 h-4" />
                    </div>
                </Card.Footer>
            </Card.Root>
        </button>

        <!-- Admin Card -->
        <button 
            onclick={() => goto("/admin")}
            class="group relative"
        >
            <div class="absolute -inset-1 bg-gradient-to-r from-amber-500/20 to-orange-600/20 rounded-3xl blur opacity-25 group-hover:opacity-100 transition duration-500"></div>
            <Card.Root class="relative bg-card/40 border-border backdrop-blur-xl h-full transition-all duration-300 group-hover:bg-card/60 group-hover:border-amber-500/50 text-left overflow-hidden">
                <Card.Header>
                    <div class="p-3 bg-muted rounded-2xl w-fit group-hover:bg-amber-500 group-hover:text-white transition-colors duration-300">
                        <ShieldCheck class="w-6 h-6" />
                    </div>
                </Card.Header>
                <Card.Content class="space-y-2">
                    <Card.Title class="text-2xl">Audio Admin</Card.Title>
                    <Card.Description class="text-base">Login to control recording, engine settings, and channel routing.</Card.Description>
                </Card.Content>
                <Card.Footer>
                    <div class="flex items-center text-sm font-bold text-amber-500 opacity-0 group-hover:opacity-100 transition-all duration-300 translate-x-[-10px] group-hover:translate-x-0">
                        ADMIN ACCESS <ArrowRight class="ml-2 w-4 h-4" />
                    </div>
                </Card.Footer>
            </Card.Root>
        </button>
    </div>
</div>
