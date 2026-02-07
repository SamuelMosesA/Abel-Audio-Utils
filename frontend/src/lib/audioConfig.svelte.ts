import { audioState } from "./audioState.svelte";
import { fileState } from "./fileState.svelte";

class AudioConfig {
    constructor() {
        this.syncConfig();
    }

    async syncConfig() {
        await audioState.syncStatus();
    }

    async connectDevice(id: number) {
        try {
            const res = await fetch("/api/control", {
                method: "POST",
                body: JSON.stringify({ action: "connect", DeviceID: id })
            });
            if (res.ok) {
                audioState.isRunning = true;
            } else {
                const err = await res.text();
                console.error("Failed to connect device:", err);
                alert("Failed to connect: " + err);
            }
        } catch (e) {
            console.error("Failed to connect device", e);
        }
    }

    async toggleRecording() {
        const action = audioState.isRecording ? "stop" : "start";
        try {
            await fetch("/api/control", {
                method: "POST",
                body: JSON.stringify({
                    action: action
                })
            });
            // Don't update state here - let the WebSocket state update handle it
        } catch (e) {
            console.error("Failed to toggle recording", e);
        }
    }

    async updateConfig() {
        try {
            await fetch("/api/control", {
                method: "POST",
                body: JSON.stringify({
                    action: "update",
                    chL: parseInt(audioState.chL.toString()),
                    chR: parseInt(audioState.chR.toString()),
                    Boost: parseFloat(audioState.boost.toString())
                })
            });
        } catch (e) {
            console.error("Failed to update config", e);
        }
    }
}

export const audioConfig = new AudioConfig();
