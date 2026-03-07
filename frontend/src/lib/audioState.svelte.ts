export interface Device {
    id: number;
    name: string;
    inputs: number;
}

export interface TranslationSession {
    language: string;
    listeners: number;
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
    translations: TranslationSession[];
    serverUrl: string;
    ssid: string;
}

class AudioState {
    // Runes for reactive state
    isRunning = $state(false);
    isRecording = $state(false);
    wsConnected = $state(false);
    devices = $state<Device[]>([]);
    selectedDeviceId = $state(0);
    chL = $state(0);
    chR = $state(0);
    boost = $state(0);
    storageLocation = $state("");
    cloudDriveLocation = $state("");
    translations = $state<TranslationSession[]>([]);
    serverUrl = $state("");
    ssid = $state("");

    // Auth and Routing state
    isAuthenticated = $state(false);
    wasKicked = $state(false);
    currentView = $state<"landing" | "stream" | "admin">("landing");
    
    #ws: WebSocket | null = null;
    onMessage: ((dv: DataView) => void) | null = null;

    constructor() {
        this.isAuthenticated = !!localStorage.getItem("admin_password");
        this.fetchDevices();
    }

    async fetchDevices() {
        try {
            const res = await fetch("/api/devices");
            this.devices = await res.json();
        } catch (e) {
            console.error("Error loading devices", e);
        }
    }

    async syncStatus() {
        try {
            const res = await fetch("/api/status");
            if (res.status === 401) {
                this.logout();
                return;
            }
            const status: AppStatus = await res.json();
            this.isRunning = status.isRunning;
            this.isRecording = status.isRecording;
            this.chL = status.chL;
            this.chR = status.chR;
            this.boost = status.boost;
            this.selectedDeviceId = status.deviceId;
            this.storageLocation = status.storageLocation;
            this.cloudDriveLocation = status.cloudDriveLocation;
            this.translations = status.translations || [];
            this.serverUrl = status.serverUrl;
            this.ssid = status.ssid;
        } catch (e) {
            console.error("Error syncing status", e);
        }
    }

    connectWebSocket() {
        console.log("Attempting WebSocket connection...");
        if (this.#ws && (this.#ws.readyState === WebSocket.OPEN || this.#ws.readyState === WebSocket.CONNECTING)) {
            console.log("WebSocket already open or connecting, skipping.");
            return;
        }
        
        this.wasKicked = false;

        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        let url = `${protocol}${window.location.host}/ws`;
        
        const pass = localStorage.getItem("admin_password");
        if (pass) {
            url += `?pass=${pass}`;
        } else {
            console.warn("WebSocket attempted aborted: no admin password in storage");
            return;
        }

        console.log("Opening new WebSocket to:", url.split('?')[0]); // Hide pass in log
        this.#ws = new WebSocket(url);
        this.#ws.binaryType = "arraybuffer";

        this.#ws.onopen = () => {
            console.log(`WebSocket connected successfully`);
            this.wsConnected = true;
        };

        this.#ws.onmessage = (event: MessageEvent) => {
            if (event.data instanceof ArrayBuffer) {
                if (this.onMessage) {
                    const dv = new DataView(event.data);
                    this.onMessage(dv);
                }
            } else {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.type === "kickout") {
                        console.warn("Kicked out by another admin session signal received!");
                        this.wasKicked = true;
                        this.logout();
                    }
                } catch (e) {
                    // Ignore
                }
            }
        };

        this.#ws.onclose = (event) => {
            console.log(`WebSocket closed (Code: ${event.code}, Reason: ${event.reason})`);
            this.wsConnected = false;
            // Only retry if we're still in admin view and NOT kicked
            if (this.currentView === "admin" && this.isAuthenticated && !this.wasKicked) {
                console.log("Auto-reconnecting in 2s...");
                setTimeout(() => this.connectWebSocket(), 2000);
            } else {
                console.log("Not reconnecting: view=" + this.currentView + ", auth=" + this.isAuthenticated + ", kicked=" + this.wasKicked);
            }
        };

        this.#ws.onerror = (e) => {
            console.error("WebSocket error:", e);
            this.wsConnected = false;
        };
    }

    async login(password: string) {
        try {
            const res = await fetch("/api/login", {
                method: "POST",
                body: JSON.stringify({ password })
            });
            if (res.ok) {
                localStorage.setItem("admin_password", password);
                this.isAuthenticated = true;
                return true;
            }
            return false;
        } catch (e) {
            console.error("Login error", e);
            return false;
        }
    }

    logout() {
        localStorage.removeItem("admin_password");
        this.isAuthenticated = false;
        if (this.#ws) {
            this.#ws.close();
            this.#ws = null;
        }
    }
}

export const audioState = new AudioState();
