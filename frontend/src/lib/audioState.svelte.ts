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
    deviceId: number;
    chL: number;
    chR: number;
    boost: number;
    storageLocation: string;
    cloudDriveLocation: string;
}

class AudioState {
    isRunning = $state(false);
    isRecording = $state(false);
    wsConnected = $state(false);
    isPrimary = $state(false);
    devices = $state<Device[]>([]);
    selectedDeviceId = $state(0);
    chL = $state(0);
    chR = $state(0);
    boost = $state(0);
    storageLocation = $state("");
    cloudDriveLocation = $state("");

    #ws: WebSocket | null = null;
    onMessage: ((dv: DataView) => void) | null = null;

    constructor() {
        this.fetchDevices();
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
            this.chL = status.chL;
            this.chR = status.chR;
            this.boost = status.boost;
            this.selectedDeviceId = status.deviceId;
            this.storageLocation = status.storageLocation;
            this.cloudDriveLocation = status.cloudDriveLocation;
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
            // Handle both state updates and audio data
            if (event.data instanceof ArrayBuffer) {
                // Audio data - Binary Protocol Format (Little Endian):
                //   Offset  Size  Field       Type      Description
                //   ------  ----  -----       ----      -----------
                //   0       4     maxL        float32   Peak level for left channel [0.0 to 1.0]
                //   4       4     maxR        float32   Peak level for right channel [0.0 to 1.0]
                //   8+      4*N   audioData   float32[] Stereo audio samples, interleaved [L, R, L, R, ...]
                if (this.onMessage) {
                    const dv = new DataView(event.data as ArrayBuffer);
                    this.onMessage(dv);
                }
            } else {
                // State update - JSON format:
                // {
                //   "type": "state",
                //   "isRunning": bool,
                //   "isRecording": bool,
                //   "isPrimary": bool,
                //   "deviceId": int,
                //   "chL": int,
                //   "chR": int,
                //   "boost": float64,
                //   "storageLocation": string,
                //   "cloudDriveLocation": string
                // }
                try {
                    const message = JSON.parse(event.data);
                    if (message.type === "state") {
                        const oldIsRecording = this.isRecording;
                        this.isRunning = message.isRunning;
                        this.isRecording = message.isRecording;
                        this.isPrimary = message.isPrimary;
                        this.selectedDeviceId = message.deviceId;
                        this.chL = message.chL;
                        this.chR = message.chR;
                        this.boost = message.boost;
                        this.storageLocation = message.storageLocation;
                        this.cloudDriveLocation = message.cloudDriveLocation;

                        // Log recording state changes
                        if (oldIsRecording !== message.isRecording) {
                            console.log(`[STATE] Recording changed: ${oldIsRecording} -> ${message.isRecording}`);
                        }
                    } else {
                        console.warn(`[STATE] Unknown message type: ${message.type}`);
                    }
                } catch (e) {
                    console.error("Failed to parse message:", e, event.data);
                }
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

    requestPrimaryControl() {
        if (this.#ws && this.#ws.readyState === WebSocket.OPEN) {
            this.#ws.send(JSON.stringify({ type: "requestPrimary" }));
        }
    }
}

export const audioState = new AudioState();
