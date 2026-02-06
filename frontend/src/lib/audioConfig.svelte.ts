import { audioState } from "./audioState.svelte";

class AudioConfig {
    chLeft = $state(0);
    chRight = $state(1);
    boost = $state(1.0);

    constructor() {
        this.syncConfig();
    }

    async syncConfig() {
        try {
            const res = await fetch("/api/status");
            const status = await res.json();
            this.chLeft = status.chL;
            this.chRight = status.chR;
            this.boost = status.boost;
        } catch (e) {
            console.error("Error syncing config", e);
        }
    }

    async connectDevice(id: string | number) {
        try {
            const res = await fetch("/api/control", {
                method: "POST",
                body: JSON.stringify({ action: "connect", DeviceID: parseInt(id.toString()) })
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
                    action,
                    Boost: parseFloat(this.boost.toString())
                })
            });
            audioState.isRecording = !audioState.isRecording;
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
                    chL: parseInt(this.chLeft.toString()),
                    chR: parseInt(this.chRight.toString()),
                    Boost: parseFloat(this.boost.toString())
                })
            });
        } catch (e) {
            console.error("Failed to update config", e);
        }
    }
}

export const audioConfig = new AudioConfig();
