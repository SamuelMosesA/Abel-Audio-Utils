<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { audio, system, visuals, ui, files } = getAppContext();
    import { goto } from "$app/navigation";
    import Button from "../ui/Button.svelte";
    import Card from "../ui/Card.svelte";
    import MeterPanel from "./MeterPanel.svelte";
    import RecordingList from "./RecordingList.svelte";
    import TranslationAdmin from "./TranslationAdmin.svelte";
    import { fetchWithSync } from "$lib/utils/api";
    import {
        Play,
        Square,
        LogOut,
        ChevronLeft,
        RotateCw,
    } from "lucide-svelte";

    let selectedDeviceValue = $state<string>("");
    let restarting = $state(false);

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

    const handleLogout = async () => {
        await system.logout();
        goto("/");
    };

    const handleApplySettings = async () => {
        const id = selectedDeviceValue !== "" ? Number(selectedDeviceValue) : null;
        await audio.commitConfig(id);
    };

    const handleRestartEngine = async () => {
        if (!confirm("Restarting the engine briefly interrupts live audio and translation for all listeners. Continue?")) {
            return;
        }
        restarting = true;
        try {
            const res = await fetchWithSync("/api/audio/restart", { method: "POST" });
            const body = await res.json().catch(() => ({}));
            let message = body.error ?? `Restart failed (${res.status})`;
            if (res.ok) {
                message = `Engine restarted. ${(body.devices ?? []).length} devices found.`;
                message += body.reconnected ? ` Reconnected to ${body.reconnected}.` : " Select your device and commit.";
                message += body.configError ? ` Settings were not reloaded: ${body.configError}` : " Settings reloaded.";
            }
            ui.showNotification(message, "engine");
            await audio.fetchDevices();
            await audio.sync();
        } finally {
            restarting = false;
        }
    };

    const handleRecording = async () => {
        try {
            const result = await audio.toggleRecording();
            await files.fetchFiles();
            if (result.processingError) ui.showNotification(result.processingError, "Recording processing");
        } catch (error) {
            ui.showNotification(error instanceof Error ? error.message : String(error), "Recording");
        }
    };
</script>

<div class="space-y-6">
    <!-- Header -->
    <header class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-border/40">
        <div>
            <div class="flex items-center gap-3">
                <h1 class="text-2xl font-bold tracking-tight">Audio Console</h1>
                <div class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border {system.wsConnected ? 'border-primary/30 bg-primary/10 text-primary' : 'border-destructive/30 bg-destructive/10 text-destructive'}">
                    <span class="w-1.5 h-1.5 rounded-full {system.wsConnected ? 'bg-primary' : 'bg-destructive'}"></span>
                    <span>{system.wsConnected ? 'Connected' : 'Offline'}</span>
                </div>
            </div>
            <p class="text-sm text-muted-foreground mt-1">Configure audio interfaces, monitor levels, and manage recordings.</p>
        </div>

        <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" onclick={() => goto("/")}>
                <ChevronLeft class="w-4 h-4 mr-1" /> Return
            </Button>
            <Button variant="ghost" size="sm" onclick={handleLogout}>
                <LogOut class="w-4 h-4 mr-1" /> Sign Out
            </Button>
        </div>
    </header>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Right Sidebar: Monitoring & Controls (First on mobile) -->
        <div class="space-y-6 lg:order-2">
            <Card 
                title="Engine Control"
                description="Manage live broadcast recording state."
            >
                <div class="space-y-4">
                    <div class="flex items-center justify-between p-3 rounded-md border {audio.isRecording ? 'border-destructive/30 bg-destructive/10 text-destructive' : 'border-border bg-muted/20 text-muted-foreground'}">
                        <span class="text-xs font-medium">Recording Status</span>
                        <div class="flex items-center gap-2">
                            <span class="w-2 h-2 rounded-full {audio.isRecording ? 'bg-destructive animate-pulse' : 'bg-muted-foreground'}"></span>
                            <span class="text-xs font-semibold uppercase">{audio.isRecording ? "Recording Active" : "Standby"}</span>
                        </div>
                    </div>

                    <div class="grid grid-cols-2 gap-3">
                        <Button 
                            class="h-16 flex flex-col items-center justify-center gap-1 font-semibold text-xs" 
                            onclick={handleRecording}
                            disabled={audio.isRecording}
                        >
                            <Play class="w-5 h-5 fill-current" />
                            START RECORDING
                        </Button>
                        <Button 
                            variant="destructive"
                            class="h-16 flex flex-col items-center justify-center gap-1 font-semibold text-xs" 
                            onclick={handleRecording}
                            disabled={!audio.isRecording}
                        >
                            <Square class="w-5 h-5 fill-current" />
                            STOP RECORDING
                        </Button>
                    </div>
                </div>
            </Card>

            <Card 
                title="Audio Monitoring"
                description="Real-time VU meters and signal preview."
            >
                <div class="space-y-4">
                    <div class="flex items-center justify-between p-2.5 rounded-md border border-border bg-muted/20">
                        <span class="text-xs font-medium text-foreground">Audio Monitor Preview</span>
                        <input
                            type="checkbox"
                            checked={visuals.monitoring}
                            onchange={() => visuals.toggleMonitor()}
                            disabled={!audio.isRunning}
                            class="w-4 h-4 accent-primary cursor-pointer rounded"
                        />
                    </div>
                    <MeterPanel />
                </div>
            </Card>

            <TranslationAdmin />
        </div>

        <!-- Audio Engine Config & Recordings (Second on mobile) -->
        <div class="lg:col-span-2 space-y-6 lg:order-1">
            <Card 
                title="Audio Engine Configuration"
                description="Select soundcard interface and configure routing channels."
            >
                <div class="space-y-4">
                    <div class="space-y-1.5">
                        <label for="device-select" class="text-xs font-medium text-muted-foreground">Input Audio Interface</label>
                        <select 
                            id="device-select"
                            class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors text-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
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
                        <div class="space-y-1.5">
                            <label for="chL-input" class="text-xs font-medium text-muted-foreground">Channel Left</label>
                            <input 
                                id="chL-input"
                                type="number" 
                                bind:value={audio.chL}
                                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm font-mono shadow-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
                                disabled={audio.isRecording}
                            />
                        </div>
                        <div class="space-y-1.5">
                            <label for="chR-input" class="text-xs font-medium text-muted-foreground">Channel Right</label>
                            <input 
                                id="chR-input"
                                type="number" 
                                bind:value={audio.chR}
                                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm font-mono shadow-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
                                disabled={audio.isRecording}
                            />
                        </div>
                    </div>

                    <div class="space-y-1.5">
                        <label for="boost-input" class="text-xs font-medium text-muted-foreground">Digital Gain Boost (dB)</label>
                        <input 
                            id="boost-input"
                            type="number" 
                            step="0.1" 
                            bind:value={audio.boost}
                            class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm font-mono shadow-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
                            disabled={audio.isRecording}
                        />
                    </div>

                    <div class="flex flex-col sm:flex-row items-center justify-between gap-3 pt-2">
                        <Button
                            variant="outline"
                            size="sm"
                            class="w-full sm:w-auto"
                            onclick={handleRestartEngine}
                            disabled={audio.isRecording || restarting}
                        >
                            <RotateCw class="w-3.5 h-3.5 mr-1.5 {restarting ? 'animate-spin' : ''}" />
                            {restarting ? "Restarting Engine..." : "Restart Engine"}
                        </Button>

                        <Button 
                            onclick={handleApplySettings} 
                            disabled={audio.isRecording}
                            size="sm"
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
