import { audioState } from "./audioState.svelte";
import { fetchWithSync } from "./utils/api";

class AudioConfig {
    async connectDevice(id: number) {
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({ action: "connect", DeviceID: id })
        });
    }

    async toggleRecording() {
        const action = audioState.isRecording ? "stop" : "start";
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({ action })
        });
    }

    async updateConfig() {
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({
                action: "update",
                chL: parseInt(audioState.chL.toString()),
                chR: parseInt(audioState.chR.toString()),
                Boost: parseFloat(audioState.boost.toString())
            })
        });
    }

    async stopTranslation(language: string) {
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({ action: "stop_translation", language, subtitles: true })
        });
        await audioState.syncStatus();
    }

    async setGeminiMaster(enabled: boolean) {
        await fetchWithSync("/api/control", {
            method: "POST",
            body: JSON.stringify({ action: "gemini_master", Enabled: enabled })
        });
        await audioState.syncStatus();
    }
}

export const audioConfig = new AudioConfig();
