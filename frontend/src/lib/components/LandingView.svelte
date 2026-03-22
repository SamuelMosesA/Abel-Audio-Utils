<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { audio, system, ui } = getAppContext();
    import { onMount } from "svelte";
    import QRCode from "qrcode";
    import { Headphones, ShieldCheck, ArrowRight, Zap, QrCode } from "lucide-svelte";
    import { goto } from "$app/navigation";
    import SimpleCard from "./ui/SimpleCard.svelte";

    ui.currentView = "landing";
    let qrContainer: HTMLCanvasElement | null = $state(null);

    onMount(async () => {
        if (system.serverUrl && qrContainer) {
            try {
                await QRCode.toCanvas(qrContainer, system.serverUrl, {
                    width: 160,
                    margin: 2,
                    color: {
                        dark: "#ffffff",
                        light: "#00000000"
                    }
                });
            } catch (err) {
                console.error("QR Code error:", err);
            }
        }
    });
</script>

<div class="max-w-5xl mx-auto space-y-12 md:space-y-20 py-12 md:py-24 px-4 md:px-6 animate-in fade-in">
    <div class="space-y-4 md:space-y-6 text-center md:text-left">
        <div class="flex items-center justify-center md:justify-start gap-2 text-primary font-bold text-xs tracking-widest uppercase">
            <Zap class="w-3.5 h-3.5 md:w-4 md:h-4 fill-current" />
            Audio Engine v2.0
        </div>
        <h1 class="text-3xl sm:text-5xl md:text-7xl font-bold tracking-tight text-white max-w-3xl leading-[1.1] mx-auto md:mx-0">
            Professional Console <br class="hidden sm:block" />
            <span class="text-muted-foreground">Digital Recorder</span>
        </h1>
        <p class="text-muted-foreground text-sm md:text-xl max-w-2xl leading-relaxed mx-auto md:mx-0">
            High-performance recording and streaming interface for Behringer X32 and M32 consoles. Synchronized cloud storage and real-time monitoring.
        </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
        <!-- Listener Card -->
        <button 
            onclick={() => goto("/stream")}
            class="group text-left"
        >
            <SimpleCard class="hover:border-primary/40 group-hover:bg-muted/5 h-full flex flex-col justify-between space-y-8 md:space-y-10 group-active:scale-[0.99] p-6 md:p-8">
                <div class="space-y-4 md:space-y-6 flex flex-col items-center md:items-start text-center md:text-left">
                    <div class="p-3 bg-primary/10 text-primary rounded-xl w-fit group-hover:bg-primary group-hover:text-primary-foreground transition-all duration-300">
                        <Headphones class="w-7 h-7" />
                    </div>
                    <div class="space-y-2">
                        <h2 class="text-2xl font-bold text-white">Live Listener</h2>
                        <p class="text-muted-foreground text-base md:text-lg text-pretty">Join as an authenticated listener to hear the live stream feed.</p>
                    </div>
                </div>
                <div class="flex items-center justify-center md:justify-start text-xs md:text-sm font-bold text-primary tracking-widest uppercase">
                    Launch Stream <ArrowRight class="ml-2 w-4 h-4 group-hover:translate-x-1 transition-transform" />
                </div>
            </SimpleCard>
        </button>

        <!-- Admin Card -->
        <button 
            onclick={() => goto("/admin")}
            class="group text-left"
        >
            <SimpleCard class="hover:border-primary/40 group-hover:bg-muted/5 h-full flex flex-col justify-between space-y-8 md:space-y-10 group-active:scale-[0.99] p-6 md:p-8">
                <div class="space-y-4 md:space-y-6 flex flex-col items-center md:items-start text-center md:text-left">
                    <div class="p-3 bg-primary/10 text-primary rounded-xl w-fit group-hover:bg-primary group-hover:text-primary-foreground transition-all duration-300">
                        <ShieldCheck class="w-7 h-7" />
                    </div>
                    <div class="space-y-2">
                        <h2 class="text-2xl font-bold text-white">Audio Admin</h2>
                        <p class="text-muted-foreground text-base md:text-lg text-pretty">Control engine settings, manage routing, and handle all recording operations.</p>
                    </div>
                </div>
                <div class="flex items-center justify-center md:justify-start text-xs md:text-sm font-bold text-primary tracking-widest uppercase">
                    Admin Portal <ArrowRight class="ml-2 w-4 h-4 group-hover:translate-x-1 transition-transform" />
                </div>
            </SimpleCard>
        </button>
    </div>

    <!-- QR Code Section -->
    {#if system.serverUrl}
    <div class="flex flex-col items-center justify-center space-y-4 pt-12 border-t border-border/20 opacity-80 hover:opacity-100 transition-opacity">
        <div class="p-4 bg-muted/30 rounded-2xl border border-border shadow-2xl backdrop-blur-xl group relative overflow-hidden">
            <div class="absolute inset-0 bg-primary/5 opacity-0 group-hover:opacity-100 transition-opacity"></div>
            
            {#if system.ssid && system.ssid !== "N/A"}
            <div class="mb-4 flex flex-col items-center space-y-1 relative z-10 border-b border-white/5 pb-4">
                <div class="flex items-center gap-2 text-primary font-black text-xs tracking-widest uppercase">
                    <Zap class="w-3 h-3" />
                    WiFi Network
                </div>
                <span class="text-sm font-bold text-white tracking-tight">{system.ssid}</span>
            </div>
            {/if}

            <canvas bind:this={qrContainer} class="relative z-10 block"></canvas>
            <div class="mt-4 flex flex-col items-center space-y-1 relative z-10">
                <div class="flex items-center gap-2 text-primary font-black text-xs tracking-widest uppercase">
                    <QrCode class="w-3 h-3" />
                    Quick Connect
                </div>
                <span class="text-xs font-mono text-muted-foreground">{system.serverUrl}</span>
            </div>
        </div>
        <p class="text-xs text-muted-foreground uppercase tracking-widest font-bold">Scan to join the stream from your mobile device</p>
    </div>
    {/if}
</div>
