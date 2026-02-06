export interface Device {
    id: number;
    name: string;
    inputs: number;
}

export interface MeterState {
    L: number;
    R: number;
}

export interface AppStatus {
    isRunning: boolean;
    isRecording: boolean;
    chL: number;
    chR: number;
    boost: number;
}

class AudioState {
    isRunning = $state(false);
    isRecording = $state(false);
    wsConnected = $state(false);
    devices = $state<Device[]>([]);
    storageLocation = $state("");

    #ws: WebSocket | null = null;
    onMessage: ((dv: DataView) => void) | null = null;

    constructor() {
        this.fetchDevices();
        this.syncStatus();
        this.connectWebSocket();
    }

    async fetchDevices() {
        try {
            const res = await fetch("/api/devices");
            this.devices = await res.json();
            console.log("Fetched devices:", this.devices.length, this.devices);
        } catch (e) {
            console.error("Error loading devices", e);
        }
    }

    async syncStatus() {
        try {
            const res = await fetch("/api/status");
            const status: AppStatus = await res.json();
            this.isRunning = status.isRunning;
            this.isRecording = status.isRecording;
        } catch (e) {
            console.error("Error syncing status", e);
        }
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        this.#ws = new WebSocket(`${protocol}${window.location.host}/ws`);
        this.#ws.binaryType = "arraybuffer";

        this.#ws.onopen = () => {
            console.log("WebSocket connected");
            this.wsConnected = true;
        };

        this.#ws.onmessage = (event: MessageEvent) => {
            if (this.onMessage) {
                const dv = new DataView(event.data as ArrayBuffer);
                this.onMessage(dv);
            }
        };

        this.#ws.onclose = () => {
            console.log("WebSocket closed, retrying...");
            this.wsConnected = false;
            setTimeout(() => this.connectWebSocket(), 2000);
        };

        this.#ws.onerror = (e) => {
            console.error("WebSocket error:", e);
            this.wsConnected = false;
        };
    }
}

export const audioState = new AudioState();
