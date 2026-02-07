<script lang="ts">
    import { onMount } from 'svelte';
    import { audioState } from './lib/audioState.svelte';
    import { fileState } from './lib/fileState.svelte';
    import { audioConfig } from './lib/audioConfig.svelte';
    import { audioVisuals } from './lib/audioVisuals.svelte';
    import RecordingList from './lib/components/RecordingList.svelte';
    import * as Card from "$lib/components/ui/card/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import { Input } from "$lib/components/ui/input/index.js";
    import { Label } from "$lib/components/ui/label/index.js";
    import { Checkbox } from "$lib/components/ui/checkbox/index.js";
    import * as Select from "$lib/components/ui/select/index.js";
    import { Mic, Radio, Play, Square, Settings, Folder, Wifi, WifiOff } from "lucide-svelte";
    import { cn } from "$lib/utils";

    // store device selection as a string to match Select component value types
    let selectedDeviceValue = $state<string | undefined>(undefined);

    // Sync device selection both ways
    $effect(() => {
        if (audioState.selectedDeviceId > 0) {
            selectedDeviceValue = audioState.selectedDeviceId.toString();
        }
    });

    // Trigger connect when user selects a device
    $effect(() => {
        if (selectedDeviceValue && Number(selectedDeviceValue) !== audioState.selectedDeviceId) {
            handleConnect();
        }
    });

    // Refresh file list when recording stops
    $effect(() => {
        if (!audioState.isRecording) {
            fileState.fetchFiles();
        }
    });

    onMount(() => {
        // Ensure state is synced from backend on load
        audioState.syncStatus();
        audioConfig.syncConfig();
    });

    const handleConnect = async () => {
        if (!selectedDeviceValue) return;
        const id = Number(selectedDeviceValue);
        if (Number.isNaN(id)) return;
        await audioConfig.connectDevice(id);
    };

    const handleToggleRec = async () => {
        await audioConfig.toggleRecording();
    };

    const handleUpdateConfig = async () => {
        await audioConfig.updateConfig();
    };

    const handleMonitorToggle = () => {
        audioVisuals.toggleMonitor();
    };

</script>

<main class="min-h-screen bg-[#0f172a] text-slate-100 p-4 md:p-8 font-sans selection:bg-indigo-500/30">
    <div class="max-w-4xl mx-auto space-y-8">
        
        <!-- Header -->
        <header class="flex items-center justify-between">
            <div class="flex items-center gap-3">
                <div class="p-2 bg-indigo-500 rounded-xl shadow-[0_0_20px_rgba(99,102,241,0.5)]">
                    <Mic class="w-6 h-6 text-white" />
                </div>
                <h1 class="text-3xl font-bold tracking-tight bg-gradient-to-r from-white to-slate-400 bg-clip-text text-transparent">
                    X32 Recorder
                </h1>
            </div>
            
            <div class="flex items-center gap-3">
                <div class={cn(
                    "px-2 py-1 rounded-md text-[10px] font-bold border flex items-center gap-1.5 transition-all duration-300",
                    audioState.wsConnected ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-500" : "bg-red-500/10 border-red-500/30 text-red-500"
                )}>
                    {#if audioState.wsConnected}
                        <Wifi class="w-3 h-3" />
                        WS ONLINE
                    {:else}
                        <WifiOff class="w-3 h-3 text-red-400" />
                        WS OFFLINE
                    {/if}
                </div>
                <div class={cn(
                    "px-3 py-1 rounded-full text-xs font-medium border flex items-center gap-2 transition-all duration-500",
                    audioState.isRecording ? "bg-red-500/10 border-red-500/50 text-red-400 animate-pulse shadow-[0_0_15px_rgba(239,68,68,0.2)]" : 
                    audioState.isRunning ? "bg-emerald-500/10 border-emerald-500/50 text-emerald-400" :
                    "bg-slate-800/50 border-slate-700 text-slate-500"
                )}>
                    {#if audioState.isRecording}
                        <span class="w-2 h-2 rounded-full bg-red-500 shadow-[0_0_8px_rgba(239,68,68,1)]"></span>
                        RECORDING
                    {:else if audioState.isRunning}
                        <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
                        ENGINE ACTIVE
                    {:else}
                        <span class="w-2 h-2 rounded-full bg-slate-600"></span>
                        STANDBY
                    {/if}
                </div>
            </div>
        </header>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            
            <!-- Connection Card -->
            <Card.Root class={cn(
                "bg-slate-900/40 border-slate-800 backdrop-blur-xl shadow-2xl overflow-hidden group",
                !audioState.isPrimary && "opacity-50 pointer-events-none"
            )}>
                <Card.Header>
                    <Card.Title class="flex items-center justify-between text-slate-300">
                        <div class="flex items-center gap-2">
                            <Radio class="w-4 h-4 text-indigo-400" />
                            Audio Interface
                        </div>
                        {#if audioState.isPrimary}
                            <span class="text-xs bg-amber-500/20 text-amber-400 px-2 py-1 rounded border border-amber-500/30 font-semibold">Primary</span>
                        {:else}
                            <span class="text-xs bg-slate-600/20 text-slate-400 px-2 py-1 rounded border border-slate-600/30 font-semibold">Secondary</span>
                        {/if}
                    </Card.Title>
                </Card.Header>
                <Card.Content class="space-y-4">
                    <div class="space-y-2">
                        <Label for="device" class="text-slate-500">Primary Device</Label>
                        <Select.Root type="single" bind:value={selectedDeviceValue} disabled={audioState.isRecording || !audioState.isPrimary}>
                            <Select.Trigger class="bg-slate-950/50 border-slate-800 text-slate-200 focus:ring-indigo-500/50">
                                {audioState.devices.find(d => d.id === Number(selectedDeviceValue))?.name ?? "Select an interface..."}
                            </Select.Trigger>
                            <Select.Content class="bg-slate-900 border-slate-800 text-slate-200">
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
            <Card.Root class={cn(
                "bg-slate-900/40 border-slate-800 backdrop-blur-xl shadow-2xl overflow-hidden",
                !audioState.isPrimary && "opacity-50 pointer-events-none"
            )}>
                <Card.Header>
                    <Card.Title class="flex items-center justify-between text-slate-300">
                        <div class="flex items-center gap-2">
                            <Settings class="w-4 h-4 text-indigo-400" />
                            Configuration
                        </div>
                        {#if !audioState.isPrimary}
                            <span class="text-xs bg-slate-600/20 text-slate-400 px-2 py-1 rounded border border-slate-600/30 font-semibold">Read-only</span>
                        {/if}
                    </Card.Title>
                </Card.Header>
                <Card.Content class="space-y-6">
                    <div class="grid grid-cols-2 gap-4">
                        <div class="space-y-2">
                            <Label for="chL" class="text-slate-500">Left Channel</Label>
                            <Input 
                                type="number" 
                                bind:value={audioState.chL} 
                                class="bg-slate-950/50 border-slate-800 text-slate-200" 
                                disabled={audioState.isRecording || !audioState.isPrimary}
                            />
                        </div>
                        <div class="space-y-2">
                            <Label for="chR" class="text-slate-500">Right Channel</Label>
                            <Input 
                                type="number" 
                                bind:value={audioState.chR} 
                                class="bg-slate-950/50 border-slate-800 text-slate-200"
                                disabled={audioState.isRecording || !audioState.isPrimary}
                            />
                        </div>
                    </div>
                    
                    <div class="space-y-2">
                        <Label for="boost" class="text-slate-500">Gain Boost</Label>
                        <Input 
                            id="boost"
                            type="number" 
                            step="0.1"
                            bind:value={audioState.boost} 
                            class="bg-slate-950/50 border-slate-800 text-slate-200" 
                            disabled={audioState.isRecording || !audioState.isPrimary}
                        />
                    </div>

                    <Button 
                        variant="secondary" 
                        class="w-full bg-slate-800 hover:bg-slate-700 text-slate-200 border-slate-700"
                        onclick={handleUpdateConfig}
                        disabled={audioState.isRecording || !audioState.isPrimary}
                    >
                        Apply Settings
                    </Button>
                </Card.Content>
            </Card.Root>

        </div>

        <!-- Transport & Monitoring -->
        <Card.Root class="bg-slate-900/40 border-slate-800 backdrop-blur-xl shadow-2xl">
            <Card.Content class="pt-6 space-y-8">
                
                <!-- Main Controls -->
                <div class="flex flex-col md:flex-row items-center justify-between gap-6 pb-6 border-b border-slate-800/50">
                    {#if audioState.isPrimary}
                        <div class="flex gap-4">
                            <Button 
                                size="lg"
                                class="min-w-[200px] h-16 text-lg font-bold transition-all duration-500 rounded-2xl flex items-center justify-center pointer-events-auto bg-slate-100 hover:bg-white text-slate-900 shadow-xl"
                                onclick={() => audioConfig.toggleRecording()}
                                disabled={!audioState.isRunning || audioState.isRecording}
                            >
                                <Play class="mr-3 w-6 h-6 fill-current pointer-events-none" />
                                Start Recording
                            </Button>
                            <Button 
                                size="lg"
                                class="min-w-[200px] h-16 text-lg font-bold transition-all duration-500 rounded-2xl flex items-center justify-center pointer-events-auto bg-red-600 hover:bg-red-700 text-white shadow-[0_0_30px_rgba(220,38,38,0.3)]"
                                onclick={() => audioConfig.toggleRecording()}
                                disabled={!audioState.isRecording}
                            >
                                <Square class="mr-3 w-6 h-6 fill-current pointer-events-none" />
                                Stop Recording
                            </Button>
                        </div>
                    {:else}
                        <Button 
                            size="lg"
                            class="bg-amber-600 hover:bg-amber-700 text-white shadow-lg"
                            onclick={() => audioState.requestPrimaryControl()}
                        >
                            Request Control
                        </Button>
                    {/if}

                    <div class="flex flex-col items-center md:items-end gap-2">
                        <div class="flex items-center gap-3 bg-slate-950/50 p-3 rounded-xl border border-slate-800">
                            <div class="flex items-center gap-2">
                                <input 
                                    type="checkbox" 
                                    id="monitor" 
                                    checked={audioVisuals.monitoring}
                                    onchange={handleMonitorToggle}
                                    disabled={!audioState.isRunning}
                                    class="border-slate-700 rounded" 
                                />
                                <Label for="monitor" class="text-sm cursor-pointer select-none text-slate-300">
                                    Low Latency Monitoring
                                </Label>
                            </div>
                        </div>
                        <div class="flex flex-col items-center md:items-end gap-2 text-right">
                            <div class="flex items-center gap-2 text-[10px] text-slate-500 font-mono">
                                <span class="bg-slate-800 px-1.5 py-0.5 rounded text-indigo-400">STORAGE</span>
                                {fileState.storageLocation}
                            </div>
                            <div class="flex items-center gap-2 text-[10px] text-slate-500 font-mono">
                                <span class="bg-slate-800 px-1.5 py-0.5 rounded text-amber-500">CLOUD</span>
                                {fileState.cloudDriveLocation}
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Meters -->
                <div class="grid grid-cols-1 gap-6">
                    <div class="space-y-3">
                        <div class="flex justify-between items-end px-1">
                            <span class="text-xs font-bold text-slate-400">Peak Meters</span>
                        </div>
                        
                        <div class="space-y-4">
                            <!-- Left -->
                            <div class="flex items-center gap-3">
                                <span class="text-xs font-bold text-slate-500 w-4">L</span>
                                <div class="flex-1 h-3 bg-slate-950 rounded-full overflow-hidden border border-slate-800 shadow-inner p-0.5">
                                    <div
                                        class={cn(
                                            "h-full transition-all duration-75 rounded-full shadow-[0_0_10px_rgba(34,197,94,0.3)]",
                                            audioVisuals.currentMeters.L > 95 ? "bg-gradient-to-r from-emerald-500 via-yellow-400 to-red-500 shadow-red-500/20" : "bg-gradient-to-r from-emerald-600 to-emerald-400"
                                        )}
                                        style="width: {audioVisuals.currentMeters.L}%"
                                    ></div>
                                </div>
                                <span class="ml-3 text-xs text-slate-400 w-16 text-right">{audioVisuals.currentDb.L <= -99 ? '−∞' : `${audioVisuals.currentDb.L.toFixed(1)} dB`}</span>
                            </div>
                            
                            <!-- Right -->
                            <div class="flex items-center gap-3">
                                <span class="text-xs font-bold text-slate-500 w-4">R</span>
                                <div class="flex-1 h-3 bg-slate-950 rounded-full overflow-hidden border border-slate-800 shadow-inner p-0.5">
                                    <div
                                        class={cn(
                                            "h-full transition-all duration-75 rounded-full shadow-[0_0_10px_rgba(34,197,94,0.3)]",
                                            audioVisuals.currentMeters.R > 95 ? "bg-gradient-to-r from-emerald-500 via-yellow-400 to-red-500 shadow-red-500/20" : "bg-gradient-to-r from-emerald-600 to-emerald-400"
                                        )}
                                        style="width: {audioVisuals.currentMeters.R}%"
                                    ></div>
                                </div>
                                <span class="ml-3 text-xs text-slate-400 w-16 text-right">{audioVisuals.currentDb.R <= -99 ? '−∞' : `${audioVisuals.currentDb.R.toFixed(1)} dB`}</span>
                            </div>
                        </div>
                    </div>
                </div>

            </Card.Content>
        </Card.Root>

        <RecordingList />

        <footer class="text-center py-8">
            <p class="text-xs text-slate-600 font-medium uppercase tracking-[0.2em]">
                &copy; 2026 Behringer X32 Audio Engine &bull; Professional Edition
            </p>
        </footer>
    </div>
</main>

<style>
    :global(body) {
        background-color: #0f172a;
    }
</style>
