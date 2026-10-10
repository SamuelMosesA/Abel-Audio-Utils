import { getContext, setContext } from "svelte";

export interface Device {
    id: number;
    name: string;
    inputs: number;
}

export interface TranslationSession {
    language: string;
    listeners: number;
    subtitles: boolean;
}

export interface LanguageStatus {
    code: string;
    name: string;
    blocked: boolean;
    active: boolean;
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

import { fetchWithSync } from "./utils/api";

export class AudioStore {
    isRunning = $state(false);
    isRecording = $state(false);
    devices = $state<Device[]>([]);
    selectedDeviceId = $state(0);
    chL = $state(0);
    chR = $state(0);
    boost = $state(0);
    storageLocation = $state("");
    cloudDriveLocation = $state("");

    async sync() {
        try {
            const res = await fetch("/api/audio/config", { credentials: "include" });
            if (res.ok) {
                const settings = await res.json();
                this.isRunning = settings.isRunning;
                this.isRecording = settings.isRecording;
                this.chL = settings.chL;
                this.chR = settings.chR;
                this.boost = settings.boost;
                this.selectedDeviceId = settings.deviceID;
                this.storageLocation = settings.storageLocation;
                this.cloudDriveLocation = settings.cloudDriveLocation;
            }
        } catch (e) {
            console.error("Error syncing audio settings", e);
        }
    }

    async fetchDevices() {
        try {
            const res = await fetch("/api/audio/devices", { credentials: "include" });
            this.devices = await res.json();
        } catch (e) {
            console.error("Error loading devices", e);
        }
    }

    async commitConfig(id: number | null) {
        const payload: any = {
            chL: this.chL,
            chR: this.chR,
            boost: this.boost
        };
        if (id !== null) payload.deviceID = id;

        await fetchWithSync("/api/audio/config", {
            method: "PATCH",
            body: JSON.stringify(payload)
        });
        await this.sync();
    }

    async toggleRecording() {
        const action = this.isRecording ? "stop" : "start";
        const res = await fetchWithSync("/api/recordings", {
            method: "POST",
            body: JSON.stringify({ action })
        });
        const payload = await res.json();
        if (!res.ok) throw new Error(payload.error || "Could not control recording");
        await this.sync();
        return payload as { processingError?: string; jobId?: string };
    }
}

export class AIStore {
    aiMasterEnabled = $state(false);
    translations = $state<TranslationSession[]>([]);
    languages = $state<LanguageStatus[]>([]);
    aiConfig = $state<{ languages: { code: string, name: string }[], originalLanguage: string }>({
        languages: [],
        originalLanguage: "en"
    });

    constructor(private ui: UIStore) {}

    async sync() {
        try {
            const res = await fetch("/api/ai/streams", { credentials: "include" });
            if (res.ok) {
                const aiData = await res.json();
                this.aiMasterEnabled = aiData.masterEnabled;
                this.translations = aiData.sessions || [];
                if (Array.isArray(aiData.languages)) {
                    this.languages = aiData.languages;
                }
            }
        } catch (e) {
            console.error("Error syncing AI status", e);
        }
    }

    async fetchConfig() {
        try {
            const res = await fetch("/api/ai/config", { credentials: "include" });
            if (res.ok) {
                this.aiConfig = await res.json();
            }
        } catch (e) {
            console.error("Error fetching AI config", e);
        }
    }

    async stopTranslation(language: string) {
        const res = await fetchWithSync("/api/ai/streams", {
            method: "POST",
            body: JSON.stringify({ action: "stop_translation", language, subtitles: true })
        });
        
        if (!res.ok) {
            const data = await res.json().catch(() => ({}));
            this.ui.showNotification(data.error || "Failed to stop translation", "error");
        }
        await this.sync();
    }

    async setAIMaster(enabled: boolean) {
        console.log("[AI] Toggling AI Master to:", enabled);
        const res = await fetchWithSync("/api/ai/streams", {
            method: "POST",
            body: JSON.stringify({ 
                action: "toggle_master", 
                enabled: enabled 
            })
        });
        
        if (!res.ok) {
            const data = await res.json().catch(() => ({}));
            this.ui.showNotification(data.error || "Failed to toggle AI", "error");
        }
        await this.sync();
    }

    async toggleLanguageKillswitch(language: string, blocked: boolean) {
        const res = await fetchWithSync("/api/ai/streams", {
            method: "POST",
            body: JSON.stringify({ 
                action: "toggle_language", 
                language, 
                blocked 
            })
        });
        
        if (!res.ok) {
            const data = await res.json().catch(() => ({}));
            this.ui.showNotification(data.error || "Failed to toggle language killswitch", "error");
        }
        await this.sync();
    }

    async refreshAIStreams() {
        await Promise.all([this.sync(), this.fetchConfig()]);
    }

    resolveLanguageName(code: string): string {
        const lang = this.aiConfig.languages.find(l => l.code === code);
        return lang ? lang.name : code;
    }
}

export class SystemStore {
    wsConnected = $state(false);
    serverUrl = $state("");
    ssid = $state("");
    isAuthenticated = $state(false);
    sessionId = $state("");

    #ws: WebSocket | null = null;
    #sse: EventSource | null = null;
    onMessage: ((dv: DataView) => void) | null = null;

    constructor(private ui: UIStore, private audio: AudioStore, private ai: AIStore, private files?: FileStore) {
        if (typeof window !== 'undefined' && window.localStorage) {
            this.sessionId = localStorage.getItem("session_id") || "";
            this.isAuthenticated = !!this.sessionId;
            
            if (this.isAuthenticated && window.location.protocol.startsWith('http')) {
                this.setupSSE();
                this.syncConnection();
            }
        }
    }

    async syncConnection() {
        try {
            const res = await fetch("/api/system/connection", { credentials: "include" });
            if (res.ok) {
                const conn = await res.json();
                this.serverUrl = conn.serverUrl;
                this.ssid = conn.ssid;
            }
        } catch (e) {
            console.error("Error syncing connection", e);
        }
    }

    setupSSE() {
        if (this.#sse) this.#sse.close();
        this.#sse = new EventSource("/api/system/changelog", { withCredentials: true });
        
        this.#sse.onmessage = (event) => {
            try {
                const change = JSON.parse(event.data);
                if (change.sessionId !== this.sessionId) {
                    this.handleRemoteUpdate(change);
                }
            } catch (e) {
                console.error("[SSE] Error parsing change:", e);
            }
        };

        this.#sse.onerror = () => {
            setTimeout(() => this.setupSSE(), 5000);
        };
    }

    handleRemoteUpdate(change: { section: string, sessionId: string }) {
        if (change.section === "ai") {
            this.ai.sync();
        } else if (change.section === "recording") {
            this.audio.sync();
            this.files?.fetchFiles();
        } else if (change.section === "interface") {
            this.audio.sync();
        } else {
            this.audio.sync();
        }
        
        this.ui.showNotification(`Session ${change.sessionId.slice(0, 4)} updated ${change.section}`, change.section);
    }

    connectWebSocket() {
        if (this.#ws && (this.#ws.readyState === WebSocket.OPEN || this.#ws.readyState === WebSocket.CONNECTING)) return;
        
        this.ui.wasKicked = false;
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        const url = `${protocol}${window.location.host}/ws`;
        
        this.#ws = new WebSocket(url);
        this.#ws.binaryType = "arraybuffer";

        this.#ws.onopen = () => this.wsConnected = true;
        this.#ws.onmessage = (event: MessageEvent) => {
            if (event.data instanceof ArrayBuffer && this.onMessage) {
                this.onMessage(new DataView(event.data));
            }
        };

        this.#ws.onclose = () => {
            this.wsConnected = false;
            if (this.ui.currentView === "admin" && this.isAuthenticated && !this.ui.wasKicked) {
                setTimeout(() => this.connectWebSocket(), 2000);
            }
        };
    }

    async login(username: string, pass: string) {
        try {
            const res = await fetch("/api/auth/session", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username, password: pass }),
                credentials: "include"
            });

            if (res.ok) {
                const data = await res.json();
                if (typeof window !== 'undefined' && window.localStorage) {
                    localStorage.setItem("admin_user", username);
                }
                if (data.session) {
                    this.sessionId = data.session;
                    if (typeof window !== 'undefined' && window.localStorage) {
                        localStorage.setItem("session_id", data.session);
                    }
                }
                this.isAuthenticated = true;
                this.setupSSE();
                this.syncConnection();
                return true;
            }
            return false;
        } catch (e) {
            console.error("Login error:", e);
            return false;
        }
    }

    async logout() {
        try {
            await fetch("/api/auth/session", {
                method: "DELETE",
                headers: { "Content-Type": "application/json" },
                credentials: "include"
            });
        } catch (e) {
            console.error("Logout request error:", e);
        }
        if (typeof window !== 'undefined' && window.localStorage) {
            localStorage.removeItem("admin_user");
            localStorage.removeItem("session_id");
        }
        this.isAuthenticated = false;
        this.sessionId = "";
        this.ui.currentView = "landing";
        if (this.#ws) { this.#ws.close(); this.#ws = null; }
        if (this.#sse) { this.#sse.close(); this.#sse = null; }
    }
}

export class UIStore {
    currentView = $state<"landing" | "stream" | "admin" | "ai_live_audio">("landing");
    notification = $state<{ message: string, section: string } | null>(null);
    wasKicked = $state(false);
    #notificationTimeout: any = null;

    showNotification(message: string, section: string) {
        if (this.#notificationTimeout) clearTimeout(this.#notificationTimeout);
        this.notification = { message, section };
        this.#notificationTimeout = setTimeout(() => {
            this.notification = null;
        }, 5000);
    }
}

import { AudioVisuals } from "./audioVisuals.svelte";

export interface ProcessingJob {
    id: string;
    source: string;
    output: string;
    stage: "queued" | "analyzing" | "encoding" | "pushing" | "completed" | "failed" | "cancelled";
    progress: number;
    processedSeconds: number;
    totalSeconds: number;
    error?: string;
    autoPush: boolean;
}

export interface RecordingExport {
    name: string;
    size: number;
    modTime: string;
    pushed: boolean;
    cloudPath: string;
    cloudTargetPath: string;
    duration: number;
}

export interface RecordedFile {
    name: string;
    display: string;
    size: number;
    modTime: string;
    duration: number;
    rawCloudPath: string;
    rawPushed: boolean;
    exports: RecordingExport[];
    jobs: ProcessingJob[];
}

export class FileStore {
    recordedFiles = $state<RecordedFile[]>([]);
    loadError = $state("");

    async fetchFiles() {
        try {
            const res = await fetch("/api/recordings/library", { credentials: "include" });
            if (res.ok) {
                const files = await res.json();
                files.sort((a: RecordedFile, b: RecordedFile) => new Date(b.modTime).getTime() - new Date(a.modTime).getTime());
                this.recordedFiles = files;
                this.loadError = "";
            } else {
                this.loadError = "Could not load recordings (HTTP " + res.status + ")";
            }
        } catch (e) {
            console.error("Error fetching files", e);
            this.loadError = "Could not connect to the recordings library";
        }
    }

    async process(source: string, startSeconds: number, endSeconds: number, automatic = false) {
        try {
            const res = await fetch("/api/recordings/process", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ source, startSeconds, endSeconds, automatic }),
                credentials: "include"
            });
            if (res.ok) { await this.fetchFiles(); return { success: true }; }
            const body = await res.json();
            return { success: false, error: body.error || res.statusText };
        } catch (e) {
            return { success: false, error: String(e) };
        }
    }

    upload(file: File, onProgress: (percent: number) => void): Promise<{ success: boolean; error?: string; processingError?: string }> {
        return new Promise(resolve => {
            const xhr = new XMLHttpRequest();
            xhr.open("POST", "/api/recordings/upload");
            xhr.withCredentials = true;
            xhr.upload.onprogress = event => {
                if (event.lengthComputable) onProgress(Math.round(event.loaded / event.total * 100));
            };
            xhr.onerror = () => resolve({ success: false, error: "Upload connection failed" });
            xhr.onload = async () => {
                let response: { error?: string; processingError?: string } = {};
                try { response = JSON.parse(xhr.responseText); } catch { /* Unexpected server response. */ }
                if (xhr.status === 201) {
                    await this.fetchFiles();
                    resolve({ success: true, processingError: response.processingError });
                } else {
                    resolve({ success: false, error: response.error || "Upload failed (HTTP " + xhr.status + ")" });
                }
            };
            const form = new FormData();
            form.append("file", file);
            xhr.send(form);
        });
    }

    async pushToCloud(source: string) {
        try {
            const res = await fetch("/api/recordings/push", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ source }),
                credentials: "include"
            });
            if (res.ok) { await this.fetchFiles(); return { success: true }; }
            const body = await res.json();
            return { success: false, error: body.error || res.statusText };
        } catch (e) {
            return { success: false, error: String(e) };
        }
    }

    async cancelProcessing(id: string) {
        try {
            const res = await fetch("/api/recordings/process/cancel", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ id }),
                credentials: "include"
            });
            if (res.ok) { await this.fetchFiles(); return { success: true }; }
            const body = await res.json();
            return { success: false, error: body.error || res.statusText };
        } catch (e) {
            return { success: false, error: String(e) };
        }
    }
}

export class AppState {
    ui = new UIStore();
    ai = new AIStore(this.ui);
    audio = new AudioStore();
    files = new FileStore();
    system: SystemStore;
    visuals: AudioVisuals;

    constructor() {
        this.system = new SystemStore(this.ui, this.audio, this.ai, this.files);
        this.visuals = new AudioVisuals(this.system);
        
        if (typeof window !== 'undefined' && window.location.protocol.startsWith('http')) {
            this.audio.fetchDevices();
            this.audio.sync();
            this.ai.fetchConfig();
            this.ai.sync();
        }
    }
}

const APP_STATE_KEY = Symbol("APP_STATE");

export function setAppContext(state: AppState) {
    return setContext(APP_STATE_KEY, state);
}

export function getAppContext() {
    return getContext<AppState>(APP_STATE_KEY);
}
