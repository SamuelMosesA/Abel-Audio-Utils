<script lang="ts">
    import { audioState } from "$lib/audioState.svelte";
    import { audioConfig } from "$lib/audioConfig.svelte";
    import { audioVisuals } from "$lib/audioVisuals.svelte";
    import { goto } from "$app/navigation";
    import * as Card from "$lib/components/ui/card/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import { Input } from "$lib/components/ui/input/index.js";
    import { Label } from "$lib/components/ui/label/index.js";
    import * as Select from "$lib/components/ui/select/index.js";
    import { Play, Square, Settings, Radio, LogOut, ChevronLeft } from "lucide-svelte";
    import MeterPanel from "./MeterPanel.svelte";
    import RecordingList from "./RecordingList.svelte";
    import { onMount } from "svelte";

    let selectedDeviceValue = $state<string | undefined>(undefined);

    onMount(() => {
        if (audioState.selectedDeviceId >= 0) {
            selectedDeviceValue = audioState.selectedDeviceId.toString();
        }
        audioState.connectWebSocket();
    });

    // Reactive sync for device selection
    $effect(() => {
        if (!selectedDeviceValue) return;
        const id = Number(selectedDeviceValue);
        if (id !== audioState.selectedDeviceId) {
            audioConfig.connectDevice(id);
        }
    });

    const handleLogout = () => {
        audioState.logout();
        goto("/");
    };

    const handleApplySettings = () => {
        audioConfig.updateConfig();
    };
</script>

<div class="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
    <!-- Admin Header -->
    <header class="flex flex-col sm:flex-row sm:items-center justify-between gap-6 pb-6 border-b border-border/50">
        <div class="flex items-center gap-4">
            <div class="p-3 bg-amber-500 rounded-2xl shadow-[0_0_20px_hsl(var(--amber-500)/0.4)]">
                <Settings class="w-7 h-7 text-white" />
            </div>
            <div>
                <h1 class="text-3xl font-black tracking-tight text-foreground">Audio Admin</h1>
                <div class="flex items-center gap-2 mt-1">
                    <div
                        class="px-2 py-0.5 rounded-md text-[9px] font-black uppercase tracking-widest border transition-all duration-300 {audioState.wsConnected ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-500' : 'bg-destructive/10 border-destructive/30 text-destructive'}"
                    >
                        {#if audioState.wsConnected}
                            <span class="inline-block w-1.5 h-1.5 rounded-full bg-emerald-500 mr-1 animate-pulse"></span>
                            WS ONLINE
                        {:else}
                            <span class="inline-block w-1.5 h-1.5 rounded-full bg-destructive mr-1"></span>
                            WS OFFLINE
                        {/if}
                    </div>
                </div>
            </div>
        </div>

        <div class="flex items-center gap-3">
            <Button 
                variant="outline" 
                onclick={() => goto("/")} 
                size="sm" 
                class="flex-1 sm:flex-none h-10 px-4 gap-2 bg-background border-2 border-border/60 hover:border-primary/50 hover:bg-primary/5 transition-all font-bold"
            >
                <ChevronLeft class="w-4 h-4" />
                Home
            </Button>
            <Button 
                variant="outline" 
                onclick={handleLogout} 
                size="sm" 
                class="flex-1 sm:flex-none h-10 px-4 gap-2 border-2 border-border/60 hover:bg-destructive/10 hover:text-destructive hover:border-destructive/30 transition-all font-bold"
            >
                <LogOut class="w-4 h-4" />
                Logout
            </Button>
        </div>
    </header>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Connection Card -->
        <Card.Root class="bg-card/40 border-border backdrop-blur-xl shadow-2xl overflow-hidden">
            <Card.Header>
                <Card.Title class="flex items-center gap-2 text-card-foreground/80">
                    <Radio class="w-4 h-4 text-primary" />
                    Audio Interface
                </Card.Title>
            </Card.Header>
            <Card.Content class="space-y-4">
                <div class="space-y-2">
                    <Label for="device" class="text-muted-foreground">Console Interface</Label>
                    <Select.Root
                        type="single"
                        bind:value={selectedDeviceValue}
                        disabled={audioState.isRecording}
                    >
                        <Select.Trigger class="bg-muted/50 border-border">
                            {audioState.devices.find((d) => d.id === Number(selectedDeviceValue))?.name ?? "Select interface..."}
                        </Select.Trigger>
                        <Select.Content class="bg-popover border-border">
                            {#each audioState.devices as device}
                                <Select.Item value={device.id.toString()} label="[{device.id}] {device.name}">
                                    [{device.id}] {device.name} ({device.inputs} in)
                                </Select.Item>
                            {/each}
                        </Select.Content>
                    </Select.Root>
                </div>
            </Card.Content>
        </Card.Root>

        <!-- Configuration Card -->
        <Card.Root class="bg-card/40 border-border backdrop-blur-xl shadow-2xl overflow-hidden">
            <Card.Header>
                <Card.Title class="flex items-center gap-2 text-card-foreground/80">
                    <Settings class="w-4 h-4 text-primary" />
                    Routing & Gain
                </Card.Title>
            </Card.Header>
            <Card.Content class="space-y-4">
                <div class="grid grid-cols-2 gap-4">
                    <div class="space-y-2">
                        <Label class="text-muted-foreground">Left Input</Label>
                        <Input type="number" bind:value={audioState.chL} class="bg-muted/50 border-border" disabled={audioState.isRecording} />
                    </div>
                    <div class="space-y-2">
                        <Label class="text-muted-foreground">Right Input</Label>
                        <Input type="number" bind:value={audioState.chR} class="bg-muted/50 border-border" disabled={audioState.isRecording} />
                    </div>
                </div>
                <div class="space-y-2">
                    <Label class="text-muted-foreground">Digital Gain Boost</Label>
                    <Input type="number" step="0.1" bind:value={audioState.boost} class="bg-muted/50 border-border" disabled={audioState.isRecording} />
                </div>
                <Button variant="secondary" class="w-full bg-secondary border-border" onclick={handleApplySettings} disabled={audioState.isRecording}>
                    Apply Settings
                </Button>
            </Card.Content>
        </Card.Root>
    </div>

    <!-- Recorder Actions (Standard Card - Side by Side) -->
    <Card.Root class="bg-card/40 border-border backdrop-blur-xl shadow-2xl overflow-hidden border-2 border-primary/20">
        <Card.Content class="py-10">
            <div class="flex flex-col items-center gap-8">
                <!-- Status Indicator Above Buttons -->
                <div class="flex flex-col items-center gap-2">
                    <div class="flex items-center gap-3 px-6 py-2 rounded-full border transition-all duration-500 {audioState.isRecording ? 'bg-destructive/10 border-destructive/30 text-destructive animate-pulse' : 'bg-muted/10 border-border/40 text-muted-foreground/40'}">
                        <span class="w-2.5 h-2.5 rounded-full {audioState.isRecording ? 'bg-destructive shadow-[0_0_12px_rgba(239,68,68,0.8)]' : 'bg-muted-foreground/30'}"></span>
                        <span class="text-[11px] font-black tracking-[0.2em] uppercase">
                            {audioState.isRecording ? 'Engine Recording Live' : 'Engine Ready'}
                        </span>
                    </div>
                </div>

                <div class="grid grid-cols-2 gap-6 w-full max-w-2xl px-4">
                    <Button
                        size="md"
                        class="text-2xl font-black rounded-3xl bg-primary text-white shadow-xl hover:bg-primary/90 transition-all duration-500 group border-4 border-transparent active:scale-95"
                        onclick={() => { if(!audioState.isRecording) audioConfig.toggleRecording(); }}
                        disabled={audioState.isRecording || !audioState.isRunning}
                    >
                        <Play class="mr-4 w-10 h-10 fill-current group-hover:scale-110 transition-transform duration-500" />
                        START
                    </Button>
                    <Button
                        size="md"
                        class="text-2xl font-black rounded-3xl bg-red-600 text-white shadow-[0_0_40px_rgba(220,38,38,0.2)] hover:bg-red-500 transition-all duration-500 group border-4 border-transparent active:scale-95"
                        onclick={() => { if(audioState.isRecording) audioConfig.toggleRecording(); }}
                        disabled={!audioState.isRecording}
                    >
                        <Square class="mr-4 w-10 h-10 fill-current group-hover:scale-110 transition-transform duration-500" />
                        STOP
                    </Button>
                </div>
            </div>
        </Card.Content>
    </Card.Root>

    <!-- Monitoring & Meters Card -->
    <Card.Root class="bg-card/40 border-border backdrop-blur-xl shadow-2xl">
        <Card.Header>
             <Card.Title class="flex items-center justify-between text-card-foreground/80">
                <div class="flex items-center gap-2">
                    <Radio class="w-4 h-4 text-primary" />
                    Live Monitoring
                </div>
                <div class="flex items-center gap-3 px-4 py-2 bg-muted/30 rounded-full border border-border/50">
                    <input
                        type="checkbox"
                        id="monitor"
                        checked={audioVisuals.monitoring}
                        onchange={() => audioVisuals.toggleMonitor()}
                        disabled={!audioState.isRunning}
                        class="w-4 h-4 rounded border-border accent-primary"
                    />
                    <Label for="monitor" class="text-xs font-bold cursor-pointer uppercase tracking-wider">Low Latency Monitor</Label>
                </div>
            </Card.Title>
        </Card.Header>
        <Card.Content class="space-y-6">
            <MeterPanel />
        </Card.Content>
    </Card.Root>

    <RecordingList />
</div>

<style>
    :global(html) {
        font-feature-settings: 'cv02', 'cv05', 'cv11', 'ss01';
    }
</style>
