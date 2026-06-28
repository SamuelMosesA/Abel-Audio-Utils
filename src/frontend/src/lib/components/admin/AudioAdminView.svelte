<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { audio, system, visuals, ui, files } = getAppContext();
    import { goto } from "$app/navigation";
    import Button from "../ui/Button.svelte";
    import Card from "../ui/Card.svelte";
    import MeterPanel from "./MeterPanel.svelte";
    import RecordingList from "./RecordingList.svelte";
    import TranslationAdmin from "./TranslationAdmin.svelte";
    import {
        Play,
        Square,
        Settings,
        Radio,
        LogOut,
        ChevronLeft,
        Activity,
    } from "lucide-svelte";

    let selectedDeviceValue = $state<string>("");

    ui.currentView = "admin";
    system.connectWebSocket();

    $effect(() => {
        if (audio.selectedDeviceId >= 0) {
            selectedDeviceValue = audio.selectedDeviceId.toString();
        } else {
            selectedDeviceValue = "";
        }
    });

    const handleDeviceChange = (e: Event) => {
        const val = (e.currentTarget as HTMLSelectElement).value;
        selectedDeviceValue = val;
    };

    const handleLogout = () => {
        system.logout();
        goto("/");
    };

    const handleApplySettings = async () => {
        const id = selectedDeviceValue !== "" ? Number(selectedDeviceValue) : null;
        await audio.commitConfig(id);
    };
</script>

<div class="max-w-screen-2xl mx-auto space-y-8 py-12 px-4 animate-in fade-in duration-500">
    <!-- Header -->
    <header class="flex flex-col md:flex-row md:items-center justify-between gap-6 pb-8 border-b border-border/40">
        <div class="flex items-center gap-4">
            <div class="p-3 bg-primary/10 rounded-2xl">
                <Activity class="w-8 h-8 text-primary" />
            </div>
            <div>
                <h1 class="text-3xl font-bold tracking-tight text-white">Console Overview</h1>
                <div class="flex items-center gap-2 mt-1">
                    <span class="flex h-2 w-2 rounded-full {system.wsConnected ? 'bg-primary' : 'bg-destructive'}"></span>
                    <span class="text-xxs font-black uppercase tracking-widest text-muted-foreground">
                        {system.wsConnected ? 'WebSocket Online' : 'WebSocket Offline'}
                    </span>
                </div>
            </div>
        </div>

        <div class="flex items-center gap-3">
            <Button variant="outline" size="sm" onclick={() => goto("/")}>
                <ChevronLeft class="w-4 h-4 mr-2" /> Return
            </Button>
            <Button variant="secondary" size="sm" onclick={handleLogout}>
                <LogOut class="w-4 h-4 mr-2" /> Sign Out
            </Button>
        </div>
    </header>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <!-- Right Sidebar: Monitoring & Controls (First on mobile) -->
        <div class="space-y-8 lg:order-2">
            <Card title="Engine Control">
                <div class="flex flex-col items-center gap-8 py-4">
                    <div class="flex items-center gap-3 px-4 py-2 rounded-full border {audio.isRecording ? 'bg-destructive/10 border-destructive/20 text-destructive' : 'bg-primary/5 border-primary/20 text-primary/60'}">
                        <span class="w-2 h-2 rounded-full {audio.isRecording ? 'bg-destructive animate-pulse' : 'bg-primary/30'}"></span>
                        <span class="text-xxs font-black uppercase tracking-widest">
                            {audio.isRecording ? "Recording Active" : "Standby Mode"}
                        </span>
                    </div>

                    <div class="grid grid-cols-2 gap-4 w-full">
                        <Button 
                            class="h-28 flex flex-col gap-2 font-black text-lg" 
                            onclick={async () => { if (!audio.isRecording) { await audio.toggleRecording(); await files.fetchFiles(); } }}
                            disabled={audio.isRecording}
                        >
                            <Play class="w-8 h-8 fill-current" />
                            START
                        </Button>
                        <Button 
                            variant="destructive"
                            class="h-28 flex flex-col gap-2 font-black text-lg" 
                            onclick={async () => { if (audio.isRecording) { await audio.toggleRecording(); await files.fetchFiles(); } }}
                            disabled={!audio.isRecording}
                        >
                            <Square class="w-8 h-8 fill-current" />
                            STOP
                        </Button>
                    </div>
                </div>
            </Card>

            <Card title="Live Monitoring">
                <div class="space-y-6">
                    <div class="flex items-center justify-between bg-muted/30 p-3 rounded-lg border border-border/40">
                        <span class="text-xxs font-black uppercase tracking-widest text-muted-foreground">Monitor Audio</span>
                        <input
                            type="checkbox"
                            checked={visuals.monitoring}
                            onchange={() => visuals.toggleMonitor()}
                            disabled={!audio.isRunning}
                            class="w-5 h-5 accent-primary cursor-pointer"
                        />
                    </div>
                    <MeterPanel />
                </div>
            </Card>

            <TranslationAdmin />
        </div>

        <!-- Audio Engine Config & Recordings (Second on mobile) -->
        <div class="lg:col-span-2 space-y-8 lg:order-1">
            <Card title="Audio Engine Configuration">
                <div class="space-y-6">
                    <div class="space-y-2">
                        <label for="device-select" class="text-xxs font-black uppercase tracking-widest text-muted-foreground ml-1">Interface Device</label>
                        <select 
                            id="device-select"
                            class="w-full bg-muted/50 border border-border rounded-lg px-4 py-3 text-sm font-bold text-white focus:ring-primary focus:border-primary disabled:opacity-50"
                            value={selectedDeviceValue}
                            onchange={handleDeviceChange}
                            disabled={audio.isRecording}
                        >
                            <option value="" disabled>Select interface...</option>
                            {#each audio.devices as device}
                                <option value={device.id.toString()}>
                                    [{device.id}] {device.name}
                                </option>
                            {/each}
                        </select>
                    </div>

                    <div class="grid grid-cols-2 gap-4">
                        <div class="space-y-2">
                            <label for="chL-input" class="text-xxs font-black uppercase tracking-widest text-muted-foreground ml-1">Channel L</label>
                            <input 
                                id="chL-input"
                                type="number" 
                                bind:value={audio.chL}
                                class="w-full bg-muted/50 border border-border rounded-lg px-4 py-3 font-mono text-lg font-bold disabled:opacity-50"
                                disabled={audio.isRecording}
                            />
                        </div>
                        <div class="space-y-2">
                            <label for="chR-input" class="text-xxs font-black uppercase tracking-widest text-muted-foreground ml-1">Channel R</label>
                            <input 
                                id="chR-input"
                                type="number" 
                                bind:value={audio.chR}
                                class="w-full bg-muted/50 border border-border rounded-lg px-4 py-3 font-mono text-lg font-bold disabled:opacity-50"
                                disabled={audio.isRecording}
                            />
                        </div>
                    </div>

                    <div class="space-y-2">
                        <label for="boost-input" class="text-xxs font-black uppercase tracking-widest text-muted-foreground ml-1">Digital Gain Boost</label>
                        <input 
                            id="boost-input"
                            type="number" 
                            step="0.1" 
                            bind:value={audio.boost}
                            class="w-full bg-muted/50 border border-border rounded-lg px-4 py-3 font-mono text-lg font-bold disabled:opacity-50"
                            disabled={audio.isRecording}
                        />
                    </div>

                    <div class="flex justify-end pt-4">
                        <Button 
                            onclick={handleApplySettings} 
                            disabled={audio.isRecording}
                            class="w-full sm:w-auto"
                        >
                            Commit Configuration
                        </Button>
                    </div>
                </div>
            </Card>

            <RecordingList />
        </div>
    </div>
</div>
