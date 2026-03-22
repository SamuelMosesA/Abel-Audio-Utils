<script lang="ts">
    import { getAppContext } from "$lib/audioState.svelte";
    const { audio, system, visuals, ui } = getAppContext();
    import { goto } from "$app/navigation";
    import { Input } from "$lib/components/ui/input/index.js";
    import { Label } from "$lib/components/ui/label/index.js";
    import * as Select from "$lib/components/ui/select/index.js";
    import {
        Play,
        Square,
        Settings,
        Radio,
        LogOut,
        ChevronLeft,
        Activity,
    } from "lucide-svelte";
    import MeterPanel from "./MeterPanel.svelte";
    import RecordingList from "./RecordingList.svelte";
    import SimpleCard from "./ui/SimpleCard.svelte";
    import SimpleButton from "./ui/SimpleButton.svelte";
    import SimpleInput from "./ui/SimpleInput.svelte";
    import TranslationAdmin from "./TranslationAdmin.svelte";
    import { onMount } from "svelte";

    let selectedDeviceValue = $state<string | undefined>(undefined);
    let isDirty = $state(false);

    ui.currentView = "admin";
    system.connectWebSocket();

    $effect(() => {
        if (audio.selectedDeviceId >= 0) {
            selectedDeviceValue = audio.selectedDeviceId.toString();
        }
    });

    $effect(() => {
        const serverId = audio.selectedDeviceId.toString();
        if (!isDirty && selectedDeviceValue !== serverId) {
            selectedDeviceValue = serverId;
        }
    });

    const handleDeviceChange = (val: string) => {
        selectedDeviceValue = val;
        isDirty = true;
    };

    const handleLogout = () => {
        system.logout();
        goto("/");
    };

    const handleApplySettings = async () => {
        const id = selectedDeviceValue ? Number(selectedDeviceValue) : null;
        await audio.commitConfig(id);
        isDirty = false;
    };
</script>

<div
    class="max-w-7xl mx-auto space-y-6 md:space-y-10 py-8 md:py-16 px-4 md:px-6 animate-in fade-in"
>
    <!-- Dashboard Header -->
    <header
        class="flex flex-col md:flex-row md:items-center justify-between gap-8 pb-10 border-b border-border/40"
    >
        <div class="flex flex-col md:flex-row items-center gap-4 md:gap-6">
            <div
                class="p-2.5 md:p-3 bg-primary/10 rounded-xl md:rounded-2xl shrink-0"
            >
                <Activity class="w-7 h-7 md:w-8 md:h-8 text-primary" />
            </div>
            <div
                class="flex flex-col items-center md:items-start space-y-2 md:space-y-1"
            >
                <h1
                    class="text-2xl md:text-4xl font-bold tracking-tight text-white leading-tight text-center md:text-left"
                >
                    Console Overview
                </h1>
                <div class="flex items-center justify-center md:justify-start gap-3">
                    <div
                        class="flex items-center gap-2 px-3 py-1 rounded-full border text-xs font-black tracking-widest uppercase {system.wsConnected
                            ? 'bg-primary/10 border-primary/20 text-primary'
                            : 'bg-destructive/10 border-destructive/20 text-destructive'}"
                    >
                        <span
                            class="w-1.5 h-1.5 rounded-full {system.wsConnected
                                ? 'bg-primary'
                                : 'bg-destructive'}"
                        ></span>
                        {system.wsConnected
                            ? "WebSocket Online"
                            : "WebSocket Offline"}
                    </div>
                </div>
            </div>
        </div>

        <div
            class="grid grid-cols-2 md:flex md:items-center gap-2 md:gap-3 w-full md:w-auto"
        >
            <SimpleButton
                onclick={() => goto("/")}
                variant="secondary"
                class="w-full"
            >
                <ChevronLeft class="w-3.5 h-3.5 md:w-4 md:h-4" />
                Return
            </SimpleButton>
            <SimpleButton
                onclick={handleLogout}
                variant="secondary"
                class="w-full hover:text-destructive hover:border-destructive/30"
            >
                <LogOut class="w-3.5 h-3.5 md:w-4 md:h-4" />
                Sign Out
            </SimpleButton>
        </div>
    </header>

    <div class="space-y-6 md:space-y-10">
        <!-- Combined Engine Parameters -->
        <SimpleCard class="space-y-6 md:space-y-8 text-white">
            <div class="flex items-center gap-3 text-muted-foreground">
                <Settings class="w-4 h-4 text-primary" />
                <span class="text-xs font-black uppercase tracking-widest"
                    >Input Audio Config</span
                >
            </div>
            
            <div class="space-y-3">
                <Label
                    for="device"
                    class="text-muted-foreground text-xs font-black tracking-wider uppercase ml-1"
                    >Portaudio Devices</Label
                >
                <Select.Root
                    type="single"
                    value={selectedDeviceValue}
                    onValueChange={handleDeviceChange}
                    disabled={audio.isRecording}
                >
                    <Select.Trigger
                        class="h-12 border-border bg-muted/50 text-white font-bold"
                    >
                        {audio.devices.find(
                            (d) => d.id === Number(selectedDeviceValue),
                        )?.name ?? "Select interface..."}
                    </Select.Trigger>
                    <Select.Content class="bg-card border-border shadow-2xl">
                        {#each audio.devices as device}
                            <Select.Item
                                value={device.id.toString()}
                                label="[{device.id}] {device.name}"
                                class="font-bold"
                            >
                                [{device.id}] {device.name}
                            </Select.Item>
                        {/each}
                    </Select.Content>
                </Select.Root>
            </div>

            <div class="grid grid-cols-2 gap-4 md:gap-6">
                <div class="space-y-3">
                    <Label
                        class="text-muted-foreground text-xs font-black tracking-wider uppercase ml-1"
                        >Ch L</Label
                    >
                    <SimpleInput
                        type="number"
                        bind:value={audio.chL}
                        class="font-mono text-lg"
                        disabled={audio.isRecording}
                    />
                </div>
                <div class="space-y-3">
                    <Label
                        class="text-muted-foreground text-xs font-black tracking-wider uppercase ml-1"
                        >Ch R</Label
                    >
                    <SimpleInput
                        type="number"
                        bind:value={audio.chR}
                        class="font-mono text-lg"
                        disabled={audio.isRecording}
                    />
                </div>
            </div>
            <div class="space-y-3">
                <Label
                    class="text-muted-foreground text-xs font-black tracking-wider uppercase ml-1"
                    >Digital Gain</Label
                >
                <div class="flex flex-col sm:flex-row gap-4">
                    <SimpleInput
                        type="number"
                        step="0.1"
                        bind:value={audio.boost}
                        class="font-mono text-lg flex-1"
                        disabled={audio.isRecording}
                    />
                </div>
            </div>

            <div class="pt-4 flex justify-end">
                <SimpleButton
                    onclick={handleApplySettings}
                    disabled={audio.isRecording}
                    class="w-full sm:w-auto px-6"
                >
                    Commit Configuration
                </SimpleButton>
            </div>
        </SimpleCard>

        <!-- Translation Monitoring -->
        <TranslationAdmin />
    </div>

    <!-- Recording Controls -->
    <div
        class="py-8 md:py-12 px-6 md:px-10 bg-card border border-primary/20 rounded-[var(--radius)] text-white transition-all"
    >
        <div class="flex flex-col items-center gap-8 md:gap-12">
            <div
                class="flex items-center gap-2 md:gap-3 px-4 md:px-6 py-1.5 md:py-2 rounded-full border {audio.isRecording
                    ? 'bg-destructive/10 border-destructive/20 text-destructive'
                    : 'bg-primary/5 border-primary/20 text-primary/60'}"
            >
                <span
                    class="w-2 md:w-2.5 h-2 md:h-2.5 rounded-full {audio.isRecording
                        ? 'bg-destructive'
                        : 'bg-primary/30'}"
                ></span>
                <span
                    class="text-xs font-black tracking-widest uppercase"
                >
                    {audio.isRecording ? "Recording" : "Standby"}
                </span>
            </div>

            <div class="grid grid-cols-2 gap-4 md:gap-8 w-full">
                <SimpleButton
                    class="h-24 md:h-32 text-xl md:text-2xl font-black rounded-xl md:rounded-2xl group flex flex-col items-center justify-center gap-1 md:gap-2"
                    onclick={() => {
                        if (!audio.isRecording)
                            audio.toggleRecording();
                    }}
                >
                    <Play
                        class="w-8 h-8 md:w-12 md:h-12 fill-current"
                    />
                    START
                </SimpleButton>
                <SimpleButton
                    variant="destructive"
                    class="h-24 md:h-32 text-xl md:text-2xl font-black rounded-xl md:rounded-2xl group flex flex-col items-center justify-center gap-1 md:gap-2"
                    onclick={() => {
                        if (audio.isRecording)
                            audio.toggleRecording();
                    }}
                >
                    <Square
                        class="w-8 h-8 md:w-12 md:h-12 fill-current"
                    />
                    STOP
                </SimpleButton>
            </div>
        </div>
    </div>

    <!-- Meters -->
    <SimpleCard class="space-y-8 md:space-y-10 text-white">
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 md:gap-3 text-muted-foreground">
                <Activity class="w-4 h-4 text-primary" />
                <span
                    class="text-xs font-black uppercase tracking-widest"
                    >Low Latency Monitors</span
                >
            </div>
            <div
                class="flex items-center gap-2 md:gap-3 px-3 md:px-4 py-1.5 md:py-2 border border-border/40 rounded-lg md:rounded-xl bg-muted/50"
            >
                <input
                    type="checkbox"
                    id="monitor"
                    checked={visuals.monitoring}
                    onchange={() => visuals.toggleMonitor()}
                    disabled={!audio.isRunning}
                    class="w-4 h-4 accent-primary rounded cursor-pointer"
                />
                <Label
                    for="monitor"
                    class="text-xs font-black cursor-pointer uppercase tracking-widest text-muted-foreground"
                    >Play Audio</Label
                >
            </div>
        </div>
        <MeterPanel />
    </SimpleCard>

    <RecordingList />
</div>
